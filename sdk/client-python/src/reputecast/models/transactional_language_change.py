from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_language_change_change import TransactionalLanguageChangeChange






T = TypeVar("T", bound="TransactionalLanguageChange")



@_attrs_define
class TransactionalLanguageChange:
    """ 
        Attributes:
            change (TransactionalLanguageChangeChange):
            language (str):
            name (str):
     """

    change: TransactionalLanguageChangeChange
    language: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        change = self.change.value

        language = self.language

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "change": change,
            "language": language,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        change = TransactionalLanguageChangeChange(d.pop("change"))




        language = d.pop("language")

        name = d.pop("name")

        transactional_language_change = cls(
            change=change,
            language=language,
            name=name,
        )

        return transactional_language_change

