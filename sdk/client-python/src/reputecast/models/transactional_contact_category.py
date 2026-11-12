from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalContactCategory")



@_attrs_define
class TransactionalContactCategory:
    """ 
        Attributes:
            id (str):
            name (str):
            on (bool):
     """

    id: str
    name: str
    on: bool





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        on = self.on


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "on": on,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        on = d.pop("on")

        transactional_contact_category = cls(
            id=id,
            name=name,
            on=on,
        )

        return transactional_contact_category

