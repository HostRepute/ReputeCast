from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalRollbackInput")



@_attrs_define
class TransactionalRollbackInput:
    """ 
        Attributes:
            version (int):
            undo (bool | Unset):
     """

    version: int
    undo: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        version = self.version

        undo = self.undo


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "version": version,
        })
        if undo is not UNSET:
            field_dict["undo"] = undo

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        version = d.pop("version")

        undo = d.pop("undo", UNSET)

        transactional_rollback_input = cls(
            version=version,
            undo=undo,
        )

        return transactional_rollback_input

