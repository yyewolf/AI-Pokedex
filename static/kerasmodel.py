import os
import requests
import numpy as np
from PIL import Image, ImageChops
from io import BytesIO
from http.server import BaseHTTPRequestHandler, HTTPServer
import tensorflow as tf
from tensorflow.keras import layers, models
import cv2
import json
from scipy import ndimage

tf.config.threading.set_inter_op_parallelism_threads(2)
tf.config.threading.set_intra_op_parallelism_threads(6)

###########################
#Loading the classic model#
###########################

classes_classic = np.load("static/classic/classes.npy", allow_pickle=True)
names_classic = np.load("static/classic/names.npy", allow_pickle=True)
ids_classic = np.load("static/classic/ids.npy", allow_pickle=True)

model_classic = tf.keras.models.load_model('static/classic/model.h5')
layers = []
for i in range(len(model_classic.layers)):
    if i != 1 and i != 2 and i != 3:
        layers.append(model_classic.layers[i])


model_classic = tf.keras.models.Sequential(layers)

###########################
#Loading the poketwo model#
###########################

classes_poketwo = np.load("static/poketwo/classes.npy", allow_pickle=True)
names_poketwo = np.load("static/poketwo/names.npy", allow_pickle=True)
ids_poketwo = np.load("static/poketwo/ids.npy", allow_pickle=True)

names_poketwo = dict(enumerate(names_poketwo.flatten(), 1))[1]
ids_poketwo = dict(enumerate(ids_poketwo.flatten(), 1))[1]

model_poketwo = tf.keras.models.load_model('static/poketwo/model.h5')
layers = []
for i in range(len(model_poketwo.layers)):
    if i != 1 and i != 2 and i != 3 and i != 4:
        layers.append(model_poketwo.layers[i])

model_poketwo = tf.keras.models.Sequential(layers)

###########################
##Finished loading models##
###########################
    
def harmonize(val, maxi, mini):
    """
        This function is used to set min and max value to a certain value.
    """
    if val < 0:
        return mini
    elif val < mini:
        return mini
    elif val > maxi:
        return int(maxi)
    return val

def center_crop(img, new_width=None, new_height=None, center=None):
    """
        This function is used to crop an image using a center point.
    """
    left, upper, right, lower = (0,0,img.size[0],img.size[1])
    if center == None:
        left = int(img.size[0]/2-new_width/2)
        upper = int(img.size[1]/2-new_height/2)
        right = left + new_width
        lower = upper + new_height
    else:
        left = int(center[0]-new_width/2)
        upper = int(center[1]-new_height/2)
        right = left + new_width
        lower = upper + new_height
        
    left = harmonize(left, img.size[0], 0)
    right = harmonize(right, img.size[0], 0)
    upper = harmonize(upper, img.size[1], 0)
    lower = harmonize(lower, img.size[1], 0)


    im_cropped = img.crop((left, upper,right,lower))
    return im_cropped

def find_center(img, th_min, th_max, blur):
    """
        This function finds the center of interest of an image using the canny method.
    """
    im = np.array(img)
    gray = cv2.cvtColor(im, cv2.COLOR_BGR2GRAY)
    blur = cv2.GaussianBlur(gray, (blur, blur), 0)
    
    canny = cv2.Canny(blur, th_min, th_max)
    
    center = ndimage.measurements.center_of_mass(canny)
    
    return (center[1], center[0])

def post_classic(img):
    width, height = img.size
    size = min(width, height)
    
    if width == 800 and height == 500:
        img = center_crop(img, width*0.7, height*0.7) # Empirical parameters found that worked best through tests
        center = find_center(img, 0, 75, 21) # Empirical parameters found that worked best through tests
        img = center_crop(img, 500, 500, (center[0], center[1]))
        width, height = img.size
        size = min(width, height)
        img = center_crop(img, size, size)
    elif width == 300 and height == 300:
        img = center_crop(img, width*0.9, height*0.9) # Empirical parameters found that worked best through tests
        center = find_center(img, 0, 50, 21) # Empirical parameters found that worked best through tests
        img = center_crop(img, 200, 200, (center[0], center[1]))

    # Optional save used for debugging
    #this_img.save('test.jpg')
    return img

def predict_this_classic(img):
    img = post_classic(img) # Prepare image
    im = img.resize((160,160)) # Size expected by network
    img_array = np.array(im)
    img_array = np.expand_dims(img_array, axis=0) # reshape from (160,160,3) to (1,160,160,3)
    pred = model_classic(img_array)
    pred = tf.keras.activations.softmax(pred)
    indexes = np.argsort(pred, axis=1)[:,-3:]
    indexes = indexes[0]
    confidences = []
    for i in indexes:
        confidences.append(pred[0][i])
    return indexes, confidences

def identify_classic(url):
    response = requests.get(url)
    if response.status_code != 200:
        return 0,0
    _img = Image.open(BytesIO(response.content))
    _img = _img.convert('RGB')
    index, conf = predict_this_classic(_img)
    return index, conf
    
def predict_this_poketwo(this_img):
    width, height = this_img.size
    if width == 800 and height == 500:
        this_img = center_crop(this_img, 800, 480)
    elif width == 300 and height == 300:
        center = find_center(this_img, 0, 50, 21) # Empirical parameters found that worked best through tests
        this_img = center_crop(this_img, 180, 180, (center[0], center[1]))
    im = this_img.resize((160,160)) # size expected by network
    img_array = np.array(im)
    img_array = np.expand_dims(img_array, axis=0) # reshape from (160,160,3) to (1,160,160,3)
    pred = model_poketwo(img_array)
    pred = tf.keras.activations.softmax(pred)
    indexes = np.argsort(pred, axis=1)[:,-3:]
    indexes = indexes[0]
    confidences = []
    for i in indexes:
        confidences.append(pred[0][i])
    return indexes, confidences

def identify_poketwo(url):
    response = requests.get(url)
    if response.status_code != 200:
        return 0,0
    _img = Image.open(BytesIO(response.content))
    _img = _img.convert('RGB')
    index, conf = predict_this_poketwo(_img)
    return index, conf

class HTTPHandler(BaseHTTPRequestHandler):
    #Handler for the POST requests
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        model_type = str(self.headers['Model'])
        url = self.rfile.read(content_length)
        predictions = []

        if model_type == "background":
            indexes, confidences = identify_poketwo(url)
            predictions = []
            for i in range(len(indexes)-1,-1,-1) : # Backward because ordered that way
                dicti = {
                    "name": names_poketwo[int(classes_poketwo[0][indexes[i]])],
                    "id": ids_poketwo[int(classes_poketwo[0][indexes[i]])],
                    "confidence": str(round(float(confidences[i]*100), 2)),
                }
                predictions.append(dicti)
        else:
            indexes, confidences = identify_classic(url)
            for i in range(len(indexes)-1,-1,-1) : # Backward because ordered that way
                dicti = {
                    "name": names_classic[int(classes_classic[0][indexes[i]])],
                    "id": ids_classic[int(classes_classic[0][indexes[i]])],
                    "confidence": str(round(float(confidences[i]*100), 2)),
                }
                predictions.append(dicti)

        self.send_response(200)
        self.send_header('Content-type','application/json')
        self.end_headers()
        # Send the html message
        self.wfile.write(
            "{"
            f"\"predictions\":{json.dumps(predictions)},"
            f"\"image url\":\"{url}\","
            f"\"model\":\"{model_type}\""
            "}".encode("utf-8")
        )

port = 5300
server = HTTPServer(('', port), HTTPHandler)

def test(urls):
    for i in urls:
        indexes, confidences = identify_poketwo(i)
        txt = ""
        for j in range(len(indexes)-1,-1,-1) :
            txt += names_poketwo[int(classes_poketwo[0][indexes[j]])]+" "
        print(txt)

urls = [
    "https://media.discordapp.net/attachments/781495172893900830/836600106332454913/pokemon.jpg", #Steelix
    "https://media.discordapp.net/attachments/781495172893900830/835782731106353162/pokemon.jpg", #Buizel
    "https://media.discordapp.net/attachments/781495172893900830/835613729370144838/pokemon.jpg", #Beldum
    "https://media.discordapp.net/attachments/781495172893900830/835613729370144838/pokemon.jpg", #Beldum
    "https://cdn.discordapp.com/attachments/781495172893900830/835617084620406854/pokemon.jpg", #Tangela
    "https://cdn.discordapp.com/attachments/781495172893900830/835720117449916426/pokemon.jpg", #Dugtrio
    "https://cdn.discordapp.com/attachments/781495172893900830/835536417496760410/pokemon.jpg", #Zubat
    "https://cdn.discordapp.com/attachments/781495172893900830/835120085424799745/pokemon.jpg", #Nidoran
    "https://media.discordapp.net/attachments/834182037501902849/837435010474573824/pokemon.jpg", #Nidoran
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
    "https://media.discordapp.net/attachments/832314403042885732/837340044707758110/pokemon.jpg", #Rufflet
    "https://media.discordapp.net/attachments/832314573041434694/837616651784552468/pokemon.jpg", #Litten
    
    "https://media.discordapp.net/attachments/834182037501902849/834338255008170064/spawn.png", #Sirfetch'd 
    "https://media.discordapp.net/attachments/834182037501902849/834313661186834492/spawn.png", #Morelull 
    "https://media.discordapp.net/attachments/594852314607517720/834383171679289344/spawn.png", #Eelektrik 
    "https://media.discordapp.net/attachments/797874693293211668/833605314829484102/spawn.png", #Metang 
    "https://media.discordapp.net/attachments/834182037501902849/834382306869116928/spawn.png", #Ambipom 
    "https://media.discordapp.net/attachments/834182037501902849/834357940584579102/spawn.png", #Glaceon 
    "https://cdn.discordapp.com/attachments/834182037501902849/837403747654303744/spawn.png", #Dubwool
    "https://cdn.discordapp.com/attachments/834182037501902849/837402089306587247/spawn.png", #Indeedee Female
    "https://cdn.discordapp.com/attachments/834182037501902849/837395803282079774/spawn.png", #Arrokuda
    "https://media.discordapp.net/attachments/834182037501902849/843819927130603560/spawn.png", #Audino
]

#test(urls)

print("Opening HTTP server")
# Wait forever for incoming http requests
server.serve_forever()