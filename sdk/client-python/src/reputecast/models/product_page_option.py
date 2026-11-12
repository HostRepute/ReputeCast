from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_page_option_status import ProductPageOptionStatus






T = TypeVar("T", bound="ProductPageOption")



@_attrs_define
class ProductPageOption:
    """ 
        Attributes:
            id (str):
            name (str):
            path (str):
            status (ProductPageOptionStatus):
     """

    id: str
    name: str
    path: str
    status: ProductPageOptionStatus





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        path = self.path

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "path": path,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        path = d.pop("path")

        status = ProductPageOptionStatus(d.pop("status"))




        product_page_option = cls(
            id=id,
            name=name,
            path=path,
            status=status,
        )

        return product_page_option

