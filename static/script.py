import aiohttp
import asyncio
import os
import csv
import random
import torch
import warnings
from io import BytesIO
from math import ceil, floor, sqrt
from PIL import Image, ImageDraw, ImageFont
from torch.autograd import Variable
from torchvision import transforms, models
from typing import Union
import sys

warnings.filterwarnings("ignore")
sys.stderr = open(os.devnull, "w")  # silence stderr

class ConditionalPad:
    def __call__(self, image):
        w, h = image.size
        if (w, h) == (800, 500):
            return image
        elif (w, h) <= (800, 500):
            wpad = (800 - w) // 2
            hpad = (500 - h) // 2
            padding = (wpad, hpad, wpad, hpad)
            padder = transforms.Pad(padding, 0, 'constant')
            return padder.__call__(image)
        else:
            resizer = transforms.Resize((800, 500))
            return resizer.__call__(image)


class PokeDetector:
    def __init__(
        self, classes_path='./static/pokeclasses.txt',
        model_path='./static/pokemodel.pth', session=None,
        old_model=False
    ):
        self.device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
        self.model = torch.load(model_path, map_location=self.device)
        self.model.eval()
        txs = [
            ConditionalPad(),
            transforms.Resize((200, 125)),
            transforms.ToTensor(),
            transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        ]
        if old_model:
            txs = [
                transforms.Resize(224),
                transforms.ToTensor(),
                transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
            ]
        self.transforms = transforms.Compose(txs)
        with open(classes_path) as f:
            self.classes = sorted(f.read().splitlines())
        self.session = session

    async def get_image_path(self, url:str)->BytesIO:
        if not self.session:
            self.session = aiohttp.ClientSession(loop=asyncio.get_event_loop())
        async with self.session.get(url) as resp:
            data = await resp.read()
        return BytesIO(data)

    def predict(self, image_path:Union[str, BytesIO])->str:
        image = Image.open(image_path).convert('RGB')
        image = self.transforms(image).float()
        image = Variable(image, requires_grad=True)
        image = image.unsqueeze(0)
        image = image.to(self.device)
        output = self.model(image)
        index = output.data.cpu().numpy().argmax()
        sm = torch.nn.Softmax()
        probabilities = sm(output) 
        return (str(self.classes[index]), probabilities[0][index].item())
    
url = 'https://media.discordapp.net/attachments/781495172893900830/833629317258280960/pokemon.jpg'
if len(sys.argv) == 2:
    url = sys.argv[1]
    
async def find(url):
    sess = aiohttp.ClientSession()
    detector = PokeDetector(
        classes_path='./static/pokeclasses.txt',
        model_path='./static/pokemodel.pth',
        session=sess
    )

    byio = await detector.get_image_path(url)
    poke, conf = detector.predict(byio)
    print(
        "{"
        f"\"name\":\"{poke}\","
        f"\"confidence\":\"{conf * 100:2.2f}%\","
        f"\"image url\":\"{url}\""
        "}"
    )
asyncio.run(find(url))