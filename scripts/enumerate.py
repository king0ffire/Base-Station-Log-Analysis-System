from enum import Enum

class State(Enum):
    """
    State of the enumeration
    """
    Noschedule=1
    Pending=2
    Running=3
    Success=4
    Failed=5
    Canceled=6
    Terminited=7