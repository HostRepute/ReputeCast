from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RecBlockUse")



@_attrs_define
class RecBlockUse:
    """ 
        Attributes:
            automation (str):
            automation_id (str):
            email_id (str):
            name (str):
     """

    automation: str
    automation_id: str
    email_id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        automation = self.automation

        automation_id = self.automation_id

        email_id = self.email_id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automation": automation,
            "automationId": automation_id,
            "emailId": email_id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automation = d.pop("automation")

        automation_id = d.pop("automationId")

        email_id = d.pop("emailId")

        name = d.pop("name")

        rec_block_use = cls(
            automation=automation,
            automation_id=automation_id,
            email_id=email_id,
            name=name,
        )

        return rec_block_use

