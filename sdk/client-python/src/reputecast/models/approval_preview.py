from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.approval_preview_language import ApprovalPreviewLanguage





T = TypeVar("T", bound="ApprovalPreview")



@_attrs_define
class ApprovalPreview:
    """ 
        Attributes:
            html (str):
            language (str):
            languages (list[ApprovalPreviewLanguage]):
            preheader (str):
            subject (str):
            version (int):
     """

    html: str
    language: str
    languages: list[ApprovalPreviewLanguage]
    preheader: str
    subject: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_preview_language import ApprovalPreviewLanguage # noqa: PLC0415
        html = self.html

        language = self.language

        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        preheader = self.preheader

        subject = self.subject

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "html": html,
            "language": language,
            "languages": languages,
            "preheader": preheader,
            "subject": subject,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_preview_language import ApprovalPreviewLanguage # noqa: PLC0415
        d = dict(src_dict)
        html = d.pop("html")

        language = d.pop("language")

        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = ApprovalPreviewLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        preheader = d.pop("preheader")

        subject = d.pop("subject")

        version = d.pop("version")

        approval_preview = cls(
            html=html,
            language=language,
            languages=languages,
            preheader=preheader,
            subject=subject,
            version=version,
        )

        return approval_preview

