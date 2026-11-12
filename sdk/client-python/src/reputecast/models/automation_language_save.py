from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationLanguageSave")



@_attrs_define
class AutomationLanguageSave:
    """ 
        Attributes:
            html (str):
            subject (str):
            version (int):
            preheader (str | Unset):
            translated (bool | Unset):
     """

    html: str
    subject: str
    version: int
    preheader: str | Unset = UNSET
    translated: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        subject = self.subject

        version = self.version

        preheader = self.preheader

        translated = self.translated


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
            "subject": subject,
            "version": version,
        })
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if translated is not UNSET:
            field_dict["translated"] = translated

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html")

        subject = d.pop("subject")

        version = d.pop("version")

        preheader = d.pop("preheader", UNSET)

        translated = d.pop("translated", UNSET)

        automation_language_save = cls(
            html=html,
            subject=subject,
            version=version,
            preheader=preheader,
            translated=translated,
        )

        return automation_language_save

