from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_email_language_status import AutomationEmailLanguageStatus






T = TypeVar("T", bound="AutomationEmailLanguage")



@_attrs_define
class AutomationEmailLanguage:
    """ 
        Attributes:
            html (str):
            language (str):
            name (str):
            preheader (str):
            status (AutomationEmailLanguageStatus):
            subject (str):
            version (int):
     """

    html: str
    language: str
    name: str
    preheader: str
    status: AutomationEmailLanguageStatus
    subject: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        language = self.language

        name = self.name

        preheader = self.preheader

        status = self.status.value

        subject = self.subject

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
            "language": language,
            "name": name,
            "preheader": preheader,
            "status": status,
            "subject": subject,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html")

        language = d.pop("language")

        name = d.pop("name")

        preheader = d.pop("preheader")

        status = AutomationEmailLanguageStatus(d.pop("status"))




        subject = d.pop("subject")

        version = d.pop("version")

        automation_email_language = cls(
            html=html,
            language=language,
            name=name,
            preheader=preheader,
            status=status,
            subject=subject,
            version=version,
        )

        return automation_email_language

