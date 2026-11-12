from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="GrowHelperSets")



@_attrs_define
class GrowHelperSets:
    """ 
        Attributes:
            connect (list[str]):
            earn (list[str]):
            edit (list[str]):
            export (list[str]):
            review (list[str]):
     """

    connect: list[str]
    earn: list[str]
    edit: list[str]
    export: list[str]
    review: list[str]





    def to_dict(self) -> dict[str, Any]:
        connect = self.connect



        earn = self.earn



        edit = self.edit



        export = self.export



        review = self.review




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connect": connect,
            "earn": earn,
            "edit": edit,
            "export": export,
            "review": review,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        connect = cast(list[str], d.pop("connect"))


        earn = cast(list[str], d.pop("earn"))


        edit = cast(list[str], d.pop("edit"))


        export = cast(list[str], d.pop("export"))


        review = cast(list[str], d.pop("review"))


        grow_helper_sets = cls(
            connect=connect,
            earn=earn,
            edit=edit,
            export=export,
            review=review,
        )

        return grow_helper_sets

