import re

#The string property returns the search string:

txt = "The rain in Spain"
x = re.search(r"S", txt)
print(x.group())
