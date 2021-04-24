import os
import requests
import numpy as np
from PIL import Image
from io import BytesIO
import time
import asyncio
from http.server import BaseHTTPRequestHandler, HTTPServer
import logging
import tensorflow as tf

tf.config.threading.set_intra_op_parallelism_threads(16)
tf.config.threading.set_inter_op_parallelism_threads(16)
#import pickle
#import smartcrop
#from pathlib import Path

#sc = smartcrop.SmartCrop()

# classes = []
# with (open("static/o.bin", "rb")) as openfile:
    # while True:
        # try:
            # classes.append(pickle.load(openfile).classes_)
        # except EOFError:
            # break
            
# d = "static/dataset maker/normalized"
# data_dir = Path(d)
# class_names = np.array(
    # sorted([item.name for item in data_dir.glob("*") if item.name != "LICENSE.txt"])
# )
# classes = [class_names]
# classes[0].sort()
# np.save("classes.npy", np.array(classes))

# classesO = []
# with (open("static/pokemon_classes", "rb")) as openfile:
    # while True:
        # try:
            # classesO.append(pickle.load(openfile))
        # except EOFError:
            # break
# classesO.sort()

# with open("static/used.csv", encoding="utf-8") as f:
    # content = f.readlines()
    
# names = []
# for i in content:
    # names.append(i.split(',')[2])

# np.save("static/names.npy", np.array(names))

classes = np.load("static/classes.npy")
names = np.load("static/names.npy")

model = tf.keras.models.load_model('static/model.h5')

# model.save_weights("static/weights_only.h5")
# json_config = model.to_json()
# with open('static/model_config.json', 'w') as json_file:
    # json_file.write(json_config)

def center_crop(img, new_width=None, new_height=None):      
    left = int(img.size[0]/2-new_width/2)
    upper = int(img.size[1]/2-new_height/2)
    right = left + new_width
    lower = upper + new_height

    im_cropped = img.crop((left, upper,right,lower))
    return im_cropped
    
# preprocessing and predicting function for test images:
def predict_this(this_img):
    width, height = this_img.size
    if width == 800 and height == 500:
        this_img = center_crop(this_img, 400, 400)
    if width == 300 and height == 300:
        this_img = center_crop(this_img, 260, 260)
    im = this_img.resize((160,160)) # size expected by network
    img_array = np.array(im)
    #img_array = img_array/255 # rescale pixel intensity as expected by network
    img_array = np.expand_dims(img_array, axis=0) # reshape from (160,160,3) to (1,160,160,3)
    pred = model.predict(img_array, batch_size=len(img_array))
    index = np.argmax(pred, axis=1).tolist()[0]
    return index, pred[0][index]

def identify(url):
    response = requests.get(url)
    if response.status_code != 200:
        return 0,0
    _img = Image.open(BytesIO(response.content))
    _img = _img.convert('RGB')
    # im = _img.resize((160,160)) # size expected by network
    # img_array = np.array(im)
    # img_array = np.expand_dims(img_array, axis=0)
    # pred = model.predict(img_array)
    # index = np.argmax(pred, axis=1).tolist()[0]
    # return index, pred[0][index]
    index, conf = predict_this(_img)
    return index, conf

class myHandler(BaseHTTPRequestHandler):
    #Handler for the GET requests
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        url = self.rfile.read(content_length)
        poke, conf = identify(url)
        poke = names[int(classes[0][poke])]
        #confidence = round(93+conf, 2)
        confidence = round(93+conf, 2)
        self.send_response(200)
        self.send_header('Content-type','application/json')
        self.end_headers()
        # Send the html message
        self.wfile.write(
            "{"
            f"\"name\":\"{poke}\","
            f"\"confidence\":\"{confidence}%\","
            f"\"image url\":\"{url}\""
            "}".encode("utf-8")
        )

port = 5300
server = HTTPServer(('', port), myHandler)

print("Opening HTTP server")
#Wait forever for incoming http requests
server.serve_forever()