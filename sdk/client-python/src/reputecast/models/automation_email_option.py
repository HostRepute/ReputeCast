from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="AutomationEmailOption")



@_attrs_define
class AutomationEmailOption:
    """ 
        Attributes:
            id (str):
            latest_version (int):
            name (str):
            subject (str):
            updated_at (datetime.datetime):
     """

    id: str
    latest_version: int
    name: str
    subject: str
    updated_at: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        latest_version = self.latest_version

        name = self.name

        subject = self.subject

        updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "latestVersion": latest_version,
            "name": name,
            "subject": subject,
            "updatedAt": updated_at,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        latest_version = d.pop("latestVersion")

        name = d.pop("name")

        subject = d.pop("subject")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        automation_email_option = cls(
            id=id,
            latest_version=latest_version,
            name=name,
            subject=subject,
            updated_at=updated_at,
        )

        return automation_email_option

