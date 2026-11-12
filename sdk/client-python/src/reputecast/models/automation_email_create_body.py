from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationEmailCreateBody")



@_attrs_define
class AutomationEmailCreateBody:
    """ 
        Attributes:
            name (str):
            automation_id (str | Unset):
            subject (str | Unset):
     """

    name: str
    automation_id: str | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        automation_id = self.automation_id

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if automation_id is not UNSET:
            field_dict["automationId"] = automation_id
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        automation_id = d.pop("automationId", UNSET)

        subject = d.pop("subject", UNSET)

        automation_email_create_body = cls(
            name=name,
            automation_id=automation_id,
            subject=subject,
        )

        return automation_email_create_body

