from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductTemplateOption")



@_attrs_define
class ProductTemplateOption:
    """ 
        Attributes:
            category (str):
            id (str):
            key (str):
            kind (str): essential or optional
            name (str):
            version (int): The live version; 0 when none is published
     """

    category: str
    id: str
    key: str
    kind: str
    name: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        category = self.category

        id = self.id

        key = self.key

        kind = self.kind

        name = self.name

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "category": category,
            "id": id,
            "key": key,
            "kind": kind,
            "name": name,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        category = d.pop("category")

        id = d.pop("id")

        key = d.pop("key")

        kind = d.pop("kind")

        name = d.pop("name")

        version = d.pop("version")

        product_template_option = cls(
            category=category,
            id=id,
            key=key,
            kind=kind,
            name=name,
            version=version,
        )

        return product_template_option

