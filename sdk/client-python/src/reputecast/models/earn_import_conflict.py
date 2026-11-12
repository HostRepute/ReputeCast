from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_import_candidate import EarnImportCandidate





T = TypeVar("T", bound="EarnImportConflict")



@_attrs_define
class EarnImportConflict:
    """ 
        Attributes:
            candidates (list[EarnImportCandidate]):
            email (str):
            row (int):
            chosen (str | Unset):
     """

    candidates: list[EarnImportCandidate]
    email: str
    row: int
    chosen: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_import_candidate import EarnImportCandidate # noqa: PLC0415
        candidates = []
        for candidates_item_data in self.candidates:
            candidates_item = candidates_item_data.to_dict()
            candidates.append(candidates_item)



        email = self.email

        row = self.row

        chosen = self.chosen


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "candidates": candidates,
            "email": email,
            "row": row,
        })
        if chosen is not UNSET:
            field_dict["chosen"] = chosen

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_import_candidate import EarnImportCandidate # noqa: PLC0415
        d = dict(src_dict)
        candidates = []
        _candidates = d.pop("candidates")
        for candidates_item_data in (_candidates):
            candidates_item = EarnImportCandidate.from_dict(candidates_item_data)



            candidates.append(candidates_item)


        email = d.pop("email")

        row = d.pop("row")

        chosen = d.pop("chosen", UNSET)

        earn_import_conflict = cls(
            candidates=candidates,
            email=email,
            row=row,
            chosen=chosen,
        )

        return earn_import_conflict

