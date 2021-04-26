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
import cv2

tf.config.threading.set_inter_op_parallelism_threads(2)
tf.config.threading.set_intra_op_parallelism_threads(6)
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
    
def harmonize(val, maxi):
    if val < 0:
        return int(0)
    elif val > maxi:
        return int(maxi)
    return val

def center_crop(img, new_width=None, new_height=None, center=None):
    left, upper, right, lower = (0,0,img.size[0],img.size[1])
    if center == None:
        left = int(img.size[0]/2-new_width/2)
        upper = int(img.size[1]/2-new_height/2)
        right = left + new_width
        lower = upper + new_height
    else:
        left = int(center[0]/2-new_width/2)
        upper = int(center[1]/2-new_height/2)
        right = left + new_width
        lower = upper + new_height
        
    left = harmonize(left, img.size[0])
    right = harmonize(right, img.size[0])
    upper = harmonize(upper, img.size[1])
    lower = harmonize(lower, img.size[1])


    im_cropped = img.crop((left, upper,right,lower))
    return im_cropped

# preprocessing and predicting function for test images:
def predict_this(this_img):
    width, height = this_img.size
    this_img2 = None
    if width == 800 and height == 500:
        this_img = center_crop(this_img, 550, 500)
        #this_img = center_crop(this_img, 800, 500)
        img = np.array(this_img)
        img = cv2.flip(img, 1)
        img = cv2.blur(img,(2,2))
        gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
        blur = cv2.GaussianBlur(gray, (9, 9), 0)
        canny = cv2.Canny(blur, 110, 190)

        ## find the non-zero min-max coords of canny
        pts = np.argwhere(canny>0)
        y1,x1 = pts.min(axis=0)
        y2,x2 = pts.max(axis=0)

        ## crop the region
        cropped = img[y1:y2, x1:x2]
        this_img = Image.fromarray(cropped)
        #this_img.save('test.jpg')
    if width == 300 and height == 300:
        this_img = center_crop(this_img, 220, 220)
        this_img2 = center_crop(this_img, 300, 250)
    im = this_img.resize((160,160)) # size expected by network
    img_array = np.array(im)
    #img_array = img_array/255 # rescale pixel intensity as expected by network
    img_array = np.expand_dims(img_array, axis=0) # reshape from (160,160,3) to (1,160,160,3)
    pred = model(img_array)
    pred = tf.keras.activations.softmax(pred)
    index = np.argmax(pred, axis=1).tolist()[0]
    if this_img2 != None:
        im2 = this_img.resize((160,160)) # size expected by network
        img_array2 = np.array(im2)
        #img_array = img_array/255 # rescale pixel intensity as expected by network
        img_array2 = np.expand_dims(img_array2, axis=0) # reshape from (160,160,3) to (1,160,160,3)
        pred2 = model(img_array2)
        pred2 = tf.keras.activations.softmax(pred2)
        index2 = np.argmax(pred2, axis=1).tolist()[0]
        if pred2[0][index2] > pred[0][index]:
            return index2, pred2[0][index2]
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
        confidence = round(float(conf*100), 2)
        #confidence = conf
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


def test(urls):
    for i in urls:
        poke, conf = identify(i)
        poke = names[int(classes[0][poke])]
        print(poke)
        
urls = [
    "https://media.discordapp.net/attachments/781495172893900830/835782731106353162/pokemon.jpg", #Buizel
    "https://media.discordapp.net/attachments/781495172893900830/835613729370144838/pokemon.jpg", #Beldum
    "https://media.discordapp.net/attachments/781495172893900830/835613729370144838/pokemon.jpg", #Beldum
    "https://cdn.discordapp.com/attachments/781495172893900830/835617084620406854/pokemon.jpg", #Tangela
    "https://cdn.discordapp.com/attachments/781495172893900830/835720117449916426/pokemon.jpg", #Dugtrio
    "https://cdn.discordapp.com/attachments/781495172893900830/835536417496760410/pokemon.jpg", #Zubat
    "https://cdn.discordapp.com/attachments/781495172893900830/835120085424799745/pokemon.jpg", #Nidoran
    "https://cdn.discordapp.com/attachments/781495172893900830/834384560690561024/pokemon.jpg", #Munna
    "https://cdn.discordapp.com/attachments/781495172893900830/834381170682101780/pokemon.jpg", #Abra
    "https://cdn.discordapp.com/attachments/781495172893900830/832983149087686677/pokemon.jpg", #Eevee
    "https://media.discordapp.net/attachments/781495172893900830/835616413682368542/pokemon.jpg", #Swablu
    "https://media.discordapp.net/attachments/781495172893900830/832024704268500992/pokemon.jpg", #Geodude
    "https://media.discordapp.net/attachments/781495172893900830/832025409855160409/pokemon.jpg", #Miltank
    "https://media.discordapp.net/attachments/781495172893900830/835820514319794226/pokemon.jpg", #Aron
    "https://media.discordapp.net/attachments/781495172893900830/835845178425868298/pokemon.jpg", #Luxray
    "https://media.discordapp.net/attachments/781495172893900830/835439388544335902/pokemon.jpg", #Grimer
    "https://media.discordapp.net/attachments/781495172893900830/835440053765406730/pokemon.jpg", #Arcanine 
    "https://media.discordapp.net/attachments/781495172893900830/835351230959845386/pokemon.jpg", #Noibat 
    
    "https://media.discordapp.net/attachments/834182037501902849/834338255008170064/spawn.png", #Sirfetch'd 
    "https://media.discordapp.net/attachments/834182037501902849/834313661186834492/spawn.png", #Morelull 
    "https://media.discordapp.net/attachments/594852314607517720/834383171679289344/spawn.png", #Eelektrik 
    "https://media.discordapp.net/attachments/797874693293211668/833605314829484102/spawn.png", #Metang 
    "https://media.discordapp.net/attachments/834182037501902849/834382306869116928/spawn.png", #Ambipom 
    "https://media.discordapp.net/attachments/834182037501902849/834357940584579102/spawn.png", #Glaceon 
]

test(urls)

port = 5300
server = HTTPServer(('', port), myHandler)

print("Opening HTTP server")
#Wait forever for incoming http requests
server.serve_forever()