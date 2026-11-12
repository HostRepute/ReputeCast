from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CommerceCollection")



@_attrs_define
class CommerceCollection:
    """ 
        Attributes:
            id (str):
            products (int):
            title (str):
     """

    id: str
    products: int
    title: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        products = self.products

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "products": products,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        products = d.pop("products")

        title = d.pop("title")

        commerce_collection = cls(
            id=id,
            products=products,
            title=title,
        )

        return commerce_collection

