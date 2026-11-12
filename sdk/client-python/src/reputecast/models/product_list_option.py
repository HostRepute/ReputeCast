from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductListOption")



@_attrs_define
class ProductListOption:
    """ 
        Attributes:
            double_opt_in (bool):
            id (str):
            name (str):
     """

    double_opt_in: bool
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        double_opt_in = self.double_opt_in

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "doubleOptIn": double_opt_in,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        double_opt_in = d.pop("doubleOptIn")

        id = d.pop("id")

        name = d.pop("name")

        product_list_option = cls(
            double_opt_in=double_opt_in,
            id=id,
            name=name,
        )

        return product_list_option

