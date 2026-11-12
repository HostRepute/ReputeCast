from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductCount")



@_attrs_define
class ProductCount:
    """ 
        Attributes:
            count (int):
            name (str):
            product_id (str):
     """

    count: int
    name: str
    product_id: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        name = self.name

        product_id = self.product_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "name": name,
            "productId": product_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        name = d.pop("name")

        product_id = d.pop("productId")

        product_count = cls(
            count=count,
            name=name,
            product_id=product_id,
        )

        return product_count

