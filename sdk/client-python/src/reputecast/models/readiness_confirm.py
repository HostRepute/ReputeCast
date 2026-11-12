from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ReadinessConfirm")



@_attrs_define
class ReadinessConfirm:
    """ 
        Attributes:
            reasons (list[str]):
            typed (bool):
            usual (int):
     """

    reasons: list[str]
    typed: bool
    usual: int





    def to_dict(self) -> dict[str, Any]:
        reasons = self.reasons



        typed = self.typed

        usual = self.usual


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "reasons": reasons,
            "typed": typed,
            "usual": usual,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        reasons = cast(list[str], d.pop("reasons"))


        typed = d.pop("typed")

        usual = d.pop("usual")

        readiness_confirm = cls(
            reasons=reasons,
            typed=typed,
            usual=usual,
        )

        return readiness_confirm

