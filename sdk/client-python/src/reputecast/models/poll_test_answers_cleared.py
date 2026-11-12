from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PollTestAnswersCleared")



@_attrs_define
class PollTestAnswersCleared:
    """ 
        Attributes:
            deleted (int):
     """

    deleted: int





    def to_dict(self) -> dict[str, Any]:
        deleted = self.deleted


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "deleted": deleted,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        deleted = d.pop("deleted")

        poll_test_answers_cleared = cls(
            deleted=deleted,
        )

        return poll_test_answers_cleared

