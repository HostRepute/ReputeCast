from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ProductsHelpers")



@_attrs_define
class ProductsHelpers:
    """ 
        Attributes:
            edit (list[str]):
            export (list[str]):
            orders (list[str]):
            publish (list[str]):
     """

    edit: list[str]
    export: list[str]
    orders: list[str]
    publish: list[str]





    def to_dict(self) -> dict[str, Any]:
        edit = self.edit



        export = self.export



        orders = self.orders



        publish = self.publish




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "edit": edit,
            "export": export,
            "orders": orders,
            "publish": publish,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        edit = cast(list[str], d.pop("edit"))


        export = cast(list[str], d.pop("export"))


        orders = cast(list[str], d.pop("orders"))


        publish = cast(list[str], d.pop("publish"))


        products_helpers = cls(
            edit=edit,
            export=export,
            orders=orders,
            publish=publish,
        )

        return products_helpers

