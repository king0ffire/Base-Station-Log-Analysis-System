import gzip
import re

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

def process_one_file_by2patterns(formatedNames,fileinfo,pattern1,pattern2):
    extractedfourEqualPattern = patterninfilebyline(fileinfo, pattern1)
    extractedfiveDashPattern = patterninfilebyline(fileinfo, pattern2)
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
    return formatedNames

def counter_FileListby2patterns(path_list,pattern1,pattern2):
    formatedNames = []
    for fileitem in path_list:
        if(fileitem[-3:]=='.gz'):
            with gzip.open(fileitem) as f:
                info = f.read().decode("utf-8")
                process_one_file_by2patterns(formatedNames,info,pattern1,pattern2)
        elif (fileitem[-4:]!='.csv'):
            with open(fileitem) as f:
                info=f.read()
                process_one_file_by2patterns(formatedNames,info,pattern1,pattern2)
    countermap=namesTocountermap(formatedNames)
    return countermap