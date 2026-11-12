from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.add_language_input_start import AddLanguageInputStart
from ..types import UNSET, Unset






T = TypeVar("T", bound="AddLanguageInput")



@_attrs_define
class AddLanguageInput:
    """ 
        Attributes:
            language (str): A workspace language
            start (AddLanguageInputStart | Unset):  Default: AddLanguageInputStart.COPY.
     """

    language: str
    start: AddLanguageInputStart | Unset = AddLanguageInputStart.COPY





    def to_dict(self) -> dict[str, Any]:
        language = self.language

        start: str | Unset = UNSET
        if not isinstance(self.start, Unset):
            start = self.start.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "language": language,
        })
        if start is not UNSET:
            field_dict["start"] = start

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        language = d.pop("language")

        _start = d.pop("start", UNSET)
        start: AddLanguageInputStart | Unset
        if isinstance(_start,  Unset):
            start = UNSET
        else:
            start = AddLanguageInputStart(_start)




        add_language_input = cls(
            language=language,
            start=start,
        )

        return add_language_input

