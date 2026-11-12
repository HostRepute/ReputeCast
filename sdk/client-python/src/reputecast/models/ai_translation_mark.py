from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AITranslationMark")



@_attrs_define
class AITranslationMark:
    """ 
        Attributes:
            language (str):
            model (str):
            reviewed (bool):
     """

    language: str
    model: str
    reviewed: bool





    def to_dict(self) -> dict[str, Any]:
        language = self.language

        model = self.model

        reviewed = self.reviewed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
            "model": model,
            "reviewed": reviewed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        language = d.pop("language")

        model = d.pop("model")

        reviewed = d.pop("reviewed")

        ai_translation_mark = cls(
            language=language,
            model=model,
            reviewed=reviewed,
        )

        return ai_translation_mark

