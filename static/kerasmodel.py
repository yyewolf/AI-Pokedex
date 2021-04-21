import os
import requests
import numpy as np
from PIL import Image
from io import BytesIO
import time
import asyncio
from http.server import BaseHTTPRequestHandler, HTTPServer
import logging
from keras.models import Model,load_model
import pickle
import smartcrop

sc = smartcrop.SmartCrop()

classes = []
with (open("static/pokemon_classes", "rb")) as openfile:
    while True:
        try:
            classes.append(pickle.load(openfile))
        except EOFError:
            break
classes.sort()

model = load_model('static/pokemon.h5')
def center_crop(img, new_width=None, new_height=None):      
    left = int(img.size[0]/2-500/2)
    upper = int(img.size[1]/2-500/2)
    right = left + 500
    lower = upper + 500

    im_cropped = img.crop((left, upper,right,lower))
    return im_cropped
    
# preprocessing and predicting function for test images:
def predict_this(this_img):
    width, height = this_img.size
    if width == 800 and height == 500:
        this_img = center_crop(this_img, 500, 500,True,0.6,0.4)
    if width == 300 and height == 300:
        r = sc.crop(this_img, 120, 120)
        this_img=this_img.crop((r["top_crop"]["x"], r["top_crop"]["y"], r["top_crop"]["x"]+r["top_crop"]["width"], r["top_crop"]["y"]+r["top_crop"]["height"]))
    im = this_img.resize((160,160)) # size expected by network
    img_array = np.array(im)
    img_array = img_array/255 # rescale pixel intensity as expected by network
    img_array = np.expand_dims(img_array, axis=0) # reshape from (160,160,3) to (1,160,160,3)
    pred = model.predict(img_array)
    index = np.argmax(pred, axis=1).tolist()[0]
    return index, pred[0][index]

def identify(url):
    response = requests.get(url)
    _img = Image.open(BytesIO(response.content)).convert('RGB')
    index, conf = predict_this(_img)
    return classes[0][index], conf

class myHandler(BaseHTTPRequestHandler):
    #Handler for the GET requests
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        url = self.rfile.read(content_length)
        poke, conf = identify(url)
        confidence = round(conf*100, 2)
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