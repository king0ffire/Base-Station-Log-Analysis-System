from pandas import read_excel

def get_category(filelocation):
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