from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GrowthRates")



@_attrs_define
class GrowthRates:
    """ 
        Attributes:
            bounce (float):
            complaint (float):
            delivered (int):
            human_open (float):
            sent (int):
     """

    bounce: float
    complaint: float
    delivered: int
    human_open: float
    sent: int





    def to_dict(self) -> dict[str, Any]:
        bounce = self.bounce

        complaint = self.complaint

        delivered = self.delivered

        human_open = self.human_open

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounce": bounce,
            "complaint": complaint,
            "delivered": delivered,
            "humanOpen": human_open,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce = d.pop("bounce")

        complaint = d.pop("complaint")

        delivered = d.pop("delivered")

        human_open = d.pop("humanOpen")

        sent = d.pop("sent")

        growth_rates = cls(
            bounce=bounce,
            complaint=complaint,
            delivered=delivered,
            human_open=human_open,
            sent=sent,
        )

        return growth_rates

