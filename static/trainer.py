import os
import requests
import numpy as np
import pandas as pd
from PIL import Image
from io import BytesIO
import matplotlib.pyplot as plt

from keras.optimizers import Adam
from keras.models import Model,load_model
from keras.applications.inception_v3 import InceptionV3
from keras.layers import Dense,Input,GlobalMaxPooling2D
from keras.preprocessing.image import ImageDataGenerator
from keras.callbacks import EarlyStopping,ReduceLROnPlateau

batch_size = 3
num_classes = 898 # this many classes of Pokemon in the dataset

data_generator = ImageDataGenerator(rescale=1./255,
                                    horizontal_flip=True,
                                    vertical_flip=False,
                                    brightness_range=(0.5,1.5),
                                    rotation_range=10,
                                    validation_split=0.2) # use the `subset` argument in `flow_from_directory` to access

train_generator = data_generator.flow_from_directory('static/pkmn',
                                                    target_size=(160,160), # chosen because this is size of the images in dataset
                                                    batch_size=batch_size,
                                                    subset='training')

val_generator = data_generator.flow_from_directory('static/pkmn',
                                                    target_size=(160,160),
                                                    batch_size=batch_size,
                                                    subset='validation')
                                                    
# import the base model and pretrained weights
custom_input = Input(shape=(160,160,3,))
base_model = InceptionV3(include_top=False, weights='imagenet', input_tensor=custom_input, input_shape=None, pooling=None, classes=num_classes)

x = base_model.layers[-1].output # snag the last layer of the imported model

x = GlobalMaxPooling2D()(x)
x = Dense(1024,activation='relu')(x) # an optional extra layer
x = Dense(num_classes,activation='softmax',name='predictions')(x) # our new, custom prediction layer

model = Model(inputs=base_model.input,outputs=x)
# new model begins from the beginning of the imported model,
# and the predictions come out of `x` (our new prediction layer)

# let's train all the layers
for layer in model.layers:
    layer.training = True
    
# these are utilities to maximize learning, while preventing over-fitting
reduce_lr = ReduceLROnPlateau(monitor='val_loss', patience=12, cooldown=6, rate=0.6, min_lr=1e-18, verbose=1)
early_stop = EarlyStopping(monitor='val_loss', patience=24, verbose=1)


model.compile(optimizer=Adam(1e-8),loss='categorical_crossentropy',metrics=['accuracy'])
model.fit(train_generator,
                    validation_data=val_generator,
                    steps_per_epoch=4,
                    validation_steps=4,
                    epochs=200, # increase this if actually training
                    shuffle=True,
                    callbacks=[reduce_lr,early_stop],
                    verbose=0)
                    
# here's how to save the model after training. Use ModelCheckpoint callback to save mid-training.
model.save('InceptionV3_Pokemon.h5')