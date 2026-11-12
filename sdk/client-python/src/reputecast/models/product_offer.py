from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductOffer")



@_attrs_define
class ProductOffer:
    """ 
        Attributes:
            list_id (str): The list a buyer who ticks it joins, with its double opt-in
            on (bool):
            text (str): The exact text, kept as consent
     """

    list_id: str
    on: bool
    text: str





    def to_dict(self) -> dict[str, Any]:
        list_id = self.list_id

        on = self.on

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listId": list_id,
            "on": on,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        list_id = d.pop("listId")

        on = d.pop("on")

        text = d.pop("text")

        product_offer = cls(
            list_id=list_id,
            on=on,
            text=text,
        )

        return product_offer

