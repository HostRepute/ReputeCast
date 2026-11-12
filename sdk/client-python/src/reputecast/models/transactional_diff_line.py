from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_diff_line_kind import TransactionalDiffLineKind






T = TypeVar("T", bound="TransactionalDiffLine")



@_attrs_define
class TransactionalDiffLine:
    """ 
        Attributes:
            kind (TransactionalDiffLineKind):
            text (str):
     """

    kind: TransactionalDiffLineKind
    text: str





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = TransactionalDiffLineKind(d.pop("kind"))




        text = d.pop("text")

        transactional_diff_line = cls(
            kind=kind,
            text=text,
        )

        return transactional_diff_line

