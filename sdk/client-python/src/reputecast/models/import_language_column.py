from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.import_language_value import ImportLanguageValue





T = TypeVar("T", bound="ImportLanguageColumn")



@_attrs_define
class ImportLanguageColumn:
    """ 
        Attributes:
            languages (list[ImportLanguageValue]):
            normalized (list[ImportLanguageValue]):
            unknown (list[ImportLanguageValue]):
            unknown_rows (int):
            unknown_values (int):
     """

    languages: list[ImportLanguageValue]
    normalized: list[ImportLanguageValue]
    unknown: list[ImportLanguageValue]
    unknown_rows: int
    unknown_values: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_language_value import ImportLanguageValue # noqa: PLC0415
        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        normalized = []
        for normalized_item_data in self.normalized:
            normalized_item = normalized_item_data.to_dict()
            normalized.append(normalized_item)



        unknown = []
        for unknown_item_data in self.unknown:
            unknown_item = unknown_item_data.to_dict()
            unknown.append(unknown_item)



        unknown_rows = self.unknown_rows

        unknown_values = self.unknown_values


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "languages": languages,
            "normalized": normalized,
            "unknown": unknown,
            "unknownRows": unknown_rows,
            "unknownValues": unknown_values,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_language_value import ImportLanguageValue # noqa: PLC0415
        d = dict(src_dict)
        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = ImportLanguageValue.from_dict(languages_item_data)



            languages.append(languages_item)


        normalized = []
        _normalized = d.pop("normalized")
        for normalized_item_data in (_normalized):
            normalized_item = ImportLanguageValue.from_dict(normalized_item_data)



            normalized.append(normalized_item)


        unknown = []
        _unknown = d.pop("unknown")
        for unknown_item_data in (_unknown):
            unknown_item = ImportLanguageValue.from_dict(unknown_item_data)



            unknown.append(unknown_item)


        unknown_rows = d.pop("unknownRows")

        unknown_values = d.pop("unknownValues")

        import_language_column = cls(
            languages=languages,
            normalized=normalized,
            unknown=unknown,
            unknown_rows=unknown_rows,
            unknown_values=unknown_values,
        )

        return import_language_column

