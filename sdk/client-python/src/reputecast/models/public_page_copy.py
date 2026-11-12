from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.public_page_copy_string import PublicPageCopyString
  from ..models.public_page_language import PublicPageLanguage





T = TypeVar("T", bound="PublicPageCopy")



@_attrs_define
class PublicPageCopy:
    """ 
        Attributes:
            copy_language (str): The language the pages are written in
            default (str):
            fallback (str):
            language (str): The language of strings (?language=)
            languages (list[PublicPageLanguage]):
            strings (list[PublicPageCopyString]):
     """

    copy_language: str
    default: str
    fallback: str
    language: str
    languages: list[PublicPageLanguage]
    strings: list[PublicPageCopyString]





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_page_copy_string import PublicPageCopyString # noqa: PLC0415
        from ..models.public_page_language import PublicPageLanguage # noqa: PLC0415
        copy_language = self.copy_language

        default = self.default

        fallback = self.fallback

        language = self.language

        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        strings = []
        for strings_item_data in self.strings:
            strings_item = strings_item_data.to_dict()
            strings.append(strings_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "copyLanguage": copy_language,
            "default": default,
            "fallback": fallback,
            "language": language,
            "languages": languages,
            "strings": strings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_page_copy_string import PublicPageCopyString # noqa: PLC0415
        from ..models.public_page_language import PublicPageLanguage # noqa: PLC0415
        d = dict(src_dict)
        copy_language = d.pop("copyLanguage")

        default = d.pop("default")

        fallback = d.pop("fallback")

        language = d.pop("language")

        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = PublicPageLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        strings = []
        _strings = d.pop("strings")
        for strings_item_data in (_strings):
            strings_item = PublicPageCopyString.from_dict(strings_item_data)



            strings.append(strings_item)


        public_page_copy = cls(
            copy_language=copy_language,
            default=default,
            fallback=fallback,
            language=language,
            languages=languages,
            strings=strings,
        )

        return public_page_copy

