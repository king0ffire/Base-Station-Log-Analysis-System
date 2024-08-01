from concurrent.futures import ThreadPoolExecutor,as_completed
import gzip
import io

import dbcount


def filenametolineiterable(file): #return : first is objects opened, second is the iterator
    objlist=[]
    if file[-3:] == ".gz":
        f= gzip.open(file)
        objlist.append(f)
        f2=io.TextIOWrapper(f,encoding='utf-8') 
        objlist.append(f2) 
        return [f,f2],f2
    elif file[-4:] != ".csv":
        f= open(file)
        objlist.append(f)
        info = f.readlines()
        return [f],info

def Parsefile(file,patternlist,countonlylist):
    openedobj,fileiterator=filenametolineiterable(file)
    formattedItems,countlist=dbcount.lineswithpattern_patternlist(fileiterator,patternlist,countonlylist)
    for obj in openedobj:
        obj.close()
    return formattedItems,countlist
    

def Parsefilelist(filelist,patternlist,countonlylist):
    formattedItems=[]
    countlist=[0 for i in range(len(countonlylist))]
    with ThreadPoolExecutor() as executor:
        fs=[executor.submit(Parsefile,file,patternlist,countonlylist) for file in filelist]
        for f in as_completed(fs):
            formattedItems.extend(f.result()[0])
            for i in range(len(countonlylist)):
                countlist[i]+=f.result()[1][i]
    return formattedItems,countlist