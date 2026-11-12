from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.email_languages import EmailLanguages





T = TypeVar("T", bound="AutomationLanguageSaved")



@_attrs_define
class AutomationLanguageSaved:
    """ 
        Attributes:
            languages (EmailLanguages):
            version (int):
     """

    languages: EmailLanguages
    version: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.email_languages import EmailLanguages # noqa: PLC0415
        languages = self.languages.to_dict()

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "languages": languages,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.email_languages import EmailLanguages # noqa: PLC0415
        d = dict(src_dict)
        languages = EmailLanguages.from_dict(d.pop("languages"))




        version = d.pop("version")

        automation_language_saved = cls(
            languages=languages,
            version=version,
        )

        return automation_language_saved

