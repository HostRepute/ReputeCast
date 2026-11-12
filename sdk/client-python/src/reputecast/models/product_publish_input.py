from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ProductPublishInput")



@_attrs_define
class ProductPublishInput:
    """ 
        Attributes:
            confirm_price (bool | Unset):
     """

    confirm_price: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        confirm_price = self.confirm_price


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if confirm_price is not UNSET:
            field_dict["confirmPrice"] = confirm_price

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        confirm_price = d.pop("confirmPrice", UNSET)

        product_publish_input = cls(
            confirm_price=confirm_price,
        )

        return product_publish_input

