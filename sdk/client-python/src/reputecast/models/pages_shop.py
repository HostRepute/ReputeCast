from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PagesShop")



@_attrs_define
class PagesShop:
    """ 
        Attributes:
            can_toggle (bool):
            on (bool):
            path (str):
            products (int):
            url (str):
     """

    can_toggle: bool
    on: bool
    path: str
    products: int
    url: str





    def to_dict(self) -> dict[str, Any]:
        can_toggle = self.can_toggle

        on = self.on

        path = self.path

        products = self.products

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canToggle": can_toggle,
            "on": on,
            "path": path,
            "products": products,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_toggle = d.pop("canToggle")

        on = d.pop("on")

        path = d.pop("path")

        products = d.pop("products")

        url = d.pop("url")

        pages_shop = cls(
            can_toggle=can_toggle,
            on=on,
            path=path,
            products=products,
            url=url,
        )

        return pages_shop

