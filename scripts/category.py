import pandas as pd

def get_category(filelocation):
    dfs = pd.read_excel(filelocation,sheet_name=None)
    category={}
    for key in dfs.keys():
        category[key]=dfs[key]['Event Name'].tolist()
    
    return category