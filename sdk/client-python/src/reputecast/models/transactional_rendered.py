from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalRendered")



@_attrs_define
class TransactionalRendered:
    """ 
        Attributes:
            html (str):
            subject (str):
            template (str):  Example: password-reset.
            text (str):
            version (int):  Example: 3.
            language (str | Unset):  Example: es.
     """

    html: str
    subject: str
    template: str
    text: str
    version: int
    language: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        subject = self.subject

        template = self.template

        text = self.text

        version = self.version

        language = self.language


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
            "subject": subject,
            "template": template,
            "text": text,
            "version": version,
        })
        if language is not UNSET:
            field_dict["language"] = language

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html")

        subject = d.pop("subject")

        template = d.pop("template")

        text = d.pop("text")

        version = d.pop("version")

        language = d.pop("language", UNSET)

        transactional_rendered = cls(
            html=html,
            subject=subject,
            template=template,
            text=text,
            version=version,
            language=language,
        )

        return transactional_rendered

