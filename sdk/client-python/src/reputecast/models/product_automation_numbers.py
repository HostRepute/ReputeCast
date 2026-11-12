from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductAutomationNumbers")



@_attrs_define
class ProductAutomationNumbers:
    """ 
        Attributes:
            contacts (int):
            entered (int): Contacts still in the flow or through it
            id (str):
            name (str):
            refunded (int): Contacts who left because their payment was refunded
     """

    contacts: int
    entered: int
    id: str
    name: str
    refunded: int





    def to_dict(self) -> dict[str, Any]:
        contacts = self.contacts

        entered = self.entered

        id = self.id

        name = self.name

        refunded = self.refunded


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "entered": entered,
            "id": id,
            "name": name,
            "refunded": refunded,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contacts = d.pop("contacts")

        entered = d.pop("entered")

        id = d.pop("id")

        name = d.pop("name")

        refunded = d.pop("refunded")

        product_automation_numbers = cls(
            contacts=contacts,
            entered=entered,
            id=id,
            name=name,
            refunded=refunded,
        )

        return product_automation_numbers

