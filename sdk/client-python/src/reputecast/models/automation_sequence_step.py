from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationSequenceStep")



@_attrs_define
class AutomationSequenceStep:
    """ 
        Attributes:
            clicked (int):
            delivered (int):
            node_id (str):
            opened_human (int):
            sent (int):
            unsubscribed (int):
            next_ (int | Unset):
     """

    clicked: int
    delivered: int
    node_id: str
    opened_human: int
    sent: int
    unsubscribed: int
    next_: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clicked = self.clicked

        delivered = self.delivered

        node_id = self.node_id

        opened_human = self.opened_human

        sent = self.sent

        unsubscribed = self.unsubscribed

        next_ = self.next_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicked": clicked,
            "delivered": delivered,
            "nodeId": node_id,
            "openedHuman": opened_human,
            "sent": sent,
            "unsubscribed": unsubscribed,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicked = d.pop("clicked")

        delivered = d.pop("delivered")

        node_id = d.pop("nodeId")

        opened_human = d.pop("openedHuman")

        sent = d.pop("sent")

        unsubscribed = d.pop("unsubscribed")

        next_ = d.pop("next", UNSET)

        automation_sequence_step = cls(
            clicked=clicked,
            delivered=delivered,
            node_id=node_id,
            opened_human=opened_human,
            sent=sent,
            unsubscribed=unsubscribed,
            next_=next_,
        )

        return automation_sequence_step

