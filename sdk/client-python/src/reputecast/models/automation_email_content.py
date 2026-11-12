from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationEmailContent")



@_attrs_define
class AutomationEmailContent:
    """ 
        Attributes:
            html (str):
            id (str):
            latest_version (int):
            name (str):
            preheader (str):
            subject (str):
            text (str):
            version (int):
     """

    html: str
    id: str
    latest_version: int
    name: str
    preheader: str
    subject: str
    text: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        id = self.id

        latest_version = self.latest_version

        name = self.name

        preheader = self.preheader

        subject = self.subject

        text = self.text

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
            "id": id,
            "latestVersion": latest_version,
            "name": name,
            "preheader": preheader,
            "subject": subject,
            "text": text,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html")

        id = d.pop("id")

        latest_version = d.pop("latestVersion")

        name = d.pop("name")

        preheader = d.pop("preheader")

        subject = d.pop("subject")

        text = d.pop("text")

        version = d.pop("version")

        automation_email_content = cls(
            html=html,
            id=id,
            latest_version=latest_version,
            name=name,
            preheader=preheader,
            subject=subject,
            text=text,
            version=version,
        )

        return automation_email_content

