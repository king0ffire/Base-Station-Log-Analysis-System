import gzip
import glob
import re
import csv
import os

log_name="Log_20240618_092153"
base_path="E:/"+log_name+"/logs/trace/trace"
path_list=glob.glob(base_path+"/dbglog*")

def patterninfile(file,pattern):
    extracted=re.findall(pattern,file)
    return extracted

def patterninfilebyline(file,pattern):
    extracted=re.findall(pattern,file,re.MULTILINE)
    return extracted
def namesTocountermap(formatedNames):
    map={}
    for item in formatedNames:
        if (item in map):
            map[item] += 1
        else:
            map[item] = 1
    return map

fourEqualPattern = r"====[^\[]*"
fiveDashPattern = r"-{5,}[^-\[\n]*"
formatedNames = []
for fileitem in path_list:
    if(fileitem[-3:]=='.gz'):
        with gzip.open(fileitem) as f:
            info = f.read().decode("utf-8")
            extractedfourEqualPattern = patterninfilebyline(info, fourEqualPattern)
            extractedfiveDashPattern = patterninfilebyline(info, fiveDashPattern)
            for i in range(len(extractedfourEqualPattern)):
                currentname=extractedfourEqualPattern[i].split(" ",1)[1].strip()
                if currentname!='':
                    formatedNames.append(currentname)
            for i in range(len(extractedfiveDashPattern)):
                currentname=extractedfiveDashPattern[i].split(" ", 1)
                if len(currentname)<=1: #面向结果编程
                    continue
                currentname=currentname[1].strip()
                if  currentname!= '':
                    formatedNames.append(currentname)
                if currentname=="18 09:20:49 info":
                    print("error")

    elif (fileitem[-4:]!='.csv'):
        with open(fileitem) as f:
            info=f.read()
            extractedfourEqualPattern=patterninfilebyline(info,fourEqualPattern)
            extractedfiveDashPattern=patterninfilebyline(info,fiveDashPattern)
            for i in range(len(extractedfourEqualPattern)):
                currentname=extractedfourEqualPattern[i].split(" ",1)[1].strip()
                if currentname!='':
                    formatedNames.append(currentname)
            for i in range(len(extractedfiveDashPattern)):
                currentname=extractedfiveDashPattern[i].split(" ",1)[1].strip()
                if currentname!='':
                    formatedNames.append(currentname)

countermap=namesTocountermap(formatedNames)
with open(os.path.join( base_path,log_name)+".csv","w",newline='') as csvfile:
    csvwriter=csv.writer(csvfile)
    for (key,value) in countermap.items():
        csvwriter.writerow([key,value])
