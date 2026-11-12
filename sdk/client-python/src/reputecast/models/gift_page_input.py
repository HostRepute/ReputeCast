from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.gift_page_input_link_months import GiftPageInputLinkMonths






T = TypeVar("T", bound="GiftPageInput")



@_attrs_define
class GiftPageInput:
    """ 
        Attributes:
            link_months (GiftPageInputLinkMonths): How long a gift link works after delivery
            on (bool):
     """

    link_months: GiftPageInputLinkMonths
    on: bool





    def to_dict(self) -> dict[str, Any]:
        link_months = self.link_months.value

        on = self.on


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "linkMonths": link_months,
            "on": on,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        link_months = GiftPageInputLinkMonths(d.pop("linkMonths"))




        on = d.pop("on")

        gift_page_input = cls(
            link_months=link_months,
            on=on,
        )

        return gift_page_input

