from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_field_choice_type import ImportFieldChoiceType
from ..types import UNSET, Unset






T = TypeVar("T", bound="ImportFieldChoice")



@_attrs_define
class ImportFieldChoice:
    """ 
        Attributes:
            source (str):
            target (str):
            type_ (ImportFieldChoiceType | Unset):
     """

    source: str
    target: str
    type_: ImportFieldChoiceType | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        source = self.source

        target = self.target

        type_: str | Unset = UNSET
        if not isinstance(self.type_, Unset):
            type_ = self.type_.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "source": source,
            "target": target,
        })
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = d.pop("source")

        target = d.pop("target")

        _type_ = d.pop("type", UNSET)
        type_: ImportFieldChoiceType | Unset
        if isinstance(_type_,  Unset):
            type_ = UNSET
        else:
            type_ = ImportFieldChoiceType(_type_)




        import_field_choice = cls(
            source=source,
            target=target,
            type_=type_,
        )

        return import_field_choice

