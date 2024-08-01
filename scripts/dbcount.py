from concurrent.futures import ThreadPoolExecutor,as_completed
import gzip
import io
import re
import logging

logger=logging.getLogger(__name__)
def patterninfile(file, pattern):
    extracted = re.findall(pattern, file)
    return extracted


def patterninfilebyline(file, pattern):
    extracted = re.findall(pattern, file, re.MULTILINE)
    return extracted


def patterninline(line, pattern):
    match_obj = re.search(pattern, line)
    return match_obj if match_obj else None


def namesTocountermap(formatedNames):
    map = {}
    for item in formatedNames:
        if item in map:
            map[item] += 1
        else:
            map[item] = 1
    return map



def process_one_file_by2patterns(formatedNames, fileinfo, pattern1, pattern2):
    extractedfourEqualPattern = patterninfilebyline(fileinfo, pattern1)
    extractedfiveDashPattern = patterninfilebyline(fileinfo, pattern2)
    for i in range(len(extractedfourEqualPattern)):
        currentname = extractedfourEqualPattern[i].split(" ", 1)[1].strip()
        if currentname != "":
            formatedNames.append(currentname)
    for i in range(len(extractedfiveDashPattern)):
        currentname = extractedfiveDashPattern[i].split(" ", 1)
        if len(currentname) <= 1:  # 面向结果编程
            continue
        currentname = currentname[1].strip()
        if currentname != "":
            formatedNames.append(currentname)
    return formatedNames



def counter_FileListby2patterns(path_list, pattern1, pattern2):
    formatedNames = []
    for fileitem in path_list:
        if fileitem[-3:] == ".gz":
            with gzip.open(fileitem) as f:
                info = f.read().decode("utf-8")
                process_one_file_by2patterns(formatedNames, info, pattern1, pattern2)
        elif fileitem[-4:] != ".csv":
            with open(fileitem) as f:
                info = f.read()
                process_one_file_by2patterns(formatedNames, info, pattern1, pattern2)
    countermap = namesTocountermap(formatedNames)
    return countermap



def ProcessFileByPattern(formatedNames, fileinfo, pattern):
    extracted = patterninfilebyline(fileinfo, pattern)
    return extracted



def ParseFiles(path_list, pattern):
    formatedNames = []
    for fileitem in path_list:
        if fileitem[-3:] == ".gz":
            with gzip.open(fileitem) as f:
                info = f.read().decode("utf-8")
                ProcessFileByPattern(formatedNames, info, pattern)
        elif fileitem[-4:] != ".csv":
            with open(fileitem) as f:
                info = f.read()
                ProcessFileByPattern(formatedNames, info, pattern)
    countermap = namesTocountermap(formatedNames)
    return countermap


def lineswithpattern(fileinfo, pattern):
    formatedItems=[]
    for line in fileinfo:
        matchobj=patterninline(line, pattern)
        if matchobj:
            line=line.strip()
            date = line[0:15].strip()
            remain = line[16:].split(" ", 1)
            errortype = remain[0].strip()
            remain = remain[1]
            remain = remain.split("|", 1)
            device = remain[0].strip()+" |"
            info = remain[1].strip() if len(remain) > 1 else ""
            extractedpattern = matchobj.group()
            eventname = extractedpattern.split(" ", 1)
            if len(eventname) <= 1:  # 面向结果编程
                    continue
            eventname = eventname[1].strip()
            if eventname != "":
                    formatedItems.append((date, errortype, device, info, eventname))
    return formatedItems


def ParseFiles_tosql_multithread(path_list, pattern):
    formatedItems = []
    objlist=[]
    fs=[]
    with ThreadPoolExecutor() as executor:
        for fileitem in path_list:
            if fileitem[-3:] == ".gz":
                f= gzip.open(fileitem)
                objlist.append(f)
                f2=io.TextIOWrapper(f,encoding='utf-8') 
                objlist.append(f2) 
                fs.append(executor.submit(lineswithpattern, f2, pattern) )
            elif fileitem[-4:] != ".csv":
                f= open(fileitem)
                objlist.append(f)
                info = f.readlines()
                fs.append(executor.submit(lineswithpattern, info, pattern) )
        for future in as_completed(fs):
            formatedItems.extend(future.result())
    for obj in objlist:
        obj.close()
    return formatedItems
        
def ParseFiles_tosql(path_list, pattern):
        formatedItems = []
        for fileitem in path_list:
            if fileitem[-3:] == ".gz":
                with gzip.open(fileitem) as f:
                    with io.TextIOWrapper(f,encoding='utf-8') as f2:
                        formatedItems.extend(lineswithpattern(f2, pattern))
            elif fileitem[-4:] != ".csv":
                with open(fileitem) as f:
                    info = f.readlines()
                    formatedItems.extend(lineswithpattern(info, pattern))
        return formatedItems




def lineswithpattern_patternlist(fileinfo, patternlist,countonlylist):
    formatedItems=[]
    countonly=[0 for i in range(len(countonlylist))]
    for line in fileinfo:
        for pattern in patternlist:
            matchobj=patterninline(line, pattern)
            if matchobj:
                line=line.strip()
                date = line[0:15].strip()
                remain = line[16:].split(" ", 1)
                errortype = remain[0].strip()
                remain = remain[1]
                remain = remain.split("|", 1)
                device = remain[0].strip()+" |"
                info = remain[1].strip() if len(remain) > 1 else ""
                extractedpattern = matchobj.group()
                eventname = extractedpattern.split(" ", 1)
                if len(eventname) <= 1:  # 面向结果编程
                        continue
                eventname = eventname[1].strip()
                if eventname != "":
                        formatedItems.append((date, errortype, device, info, eventname))
        for i in range(len(countonly)):
            matchobj=patterninline(line, countonlylist[i])
            if matchobj:
                countonly[i]+=1
            
    return formatedItems,countonly



def ParseFiles_tosql_multithread_patternlist(path_list, patternlist):
    formatedItems = []
    objlist=[]
    fs=[]
    with ThreadPoolExecutor() as executor:
        for fileitem in path_list:
            if fileitem[-3:] == ".gz":
                f= gzip.open(fileitem)
                objlist.append(f)
                f2=io.TextIOWrapper(f,encoding='utf-8') 
                objlist.append(f2) 
                fs.append(executor.submit(lineswithpattern, f2, patternlist) )
            elif fileitem[-4:] != ".csv":
                f= open(fileitem)
                objlist.append(f)
                info = f.readlines()
                fs.append(executor.submit(lineswithpattern, info, patternlist) )
        for future in as_completed(fs):
            formatedItems.extend(future.result())
    for obj in objlist:
        obj.close()
    return formatedItems