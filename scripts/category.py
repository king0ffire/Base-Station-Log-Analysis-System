
import csv
'''
from pandas import read_excel
def get_category(filelocation): #[categoryname]: list[eventname]
    dfs = read_excel(filelocation,sheet_name=None)
    category={}
    for key in dfs.keys():
        category[key]=dfs[key]['Event Name'].tolist()
    
    return category

def get_tag(countmap, categories):
    tags = {}
    for key in countmap.keys():
        tags[key]=[]
    
    for key, categorylist in categories.items():
        for category in categorylist:
                tags[category].append(key)
    return tags
'''
def get_tagfromcsv(filelocation):
    tags={}
    with open(filelocation, 'r',encoding='utf-8') as csvfile:
        reader = csv.reader(csvfile)
        for row in reader:
            if row[0]=='Event Name':
                continue
            if row[0] in tags:
                tags[row[0]].append(row[1])
            else:
                tags[row[0]]=[row[1]]
    return tags