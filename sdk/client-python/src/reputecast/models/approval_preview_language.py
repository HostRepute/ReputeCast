from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_preview_language_status import ApprovalPreviewLanguageStatus






T = TypeVar("T", bound="ApprovalPreviewLanguage")



@_attrs_define
class ApprovalPreviewLanguage:
    """ 
        Attributes:
            language (str):
            name (str):
            status (ApprovalPreviewLanguageStatus):
     """

    language: str
    name: str
    status: ApprovalPreviewLanguageStatus





    def to_dict(self) -> dict[str, Any]:
        language = self.language

        name = self.name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        language = d.pop("language")

        name = d.pop("name")

        status = ApprovalPreviewLanguageStatus(d.pop("status"))




        approval_preview_language = cls(
            language=language,
            name=name,
            status=status,
        )

        return approval_preview_language

