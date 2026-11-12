from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_check_fix_kind import TransactionalCheckFixKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TransactionalCheckFix")



@_attrs_define
class TransactionalCheckFix:
    """ 
        Attributes:
            kind (TransactionalCheckFixKind):
            label (str):
            path (list[int] | Unset):
            test_data (str | Unset):
            value (str | Unset):
            variable (str | Unset):
     """

    kind: TransactionalCheckFixKind
    label: str
    path: list[int] | Unset = UNSET
    test_data: str | Unset = UNSET
    value: str | Unset = UNSET
    variable: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        path: list[int] | Unset = UNSET
        if not isinstance(self.path, Unset):
            path = self.path



        test_data = self.test_data

        value = self.value

        variable = self.variable


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if path is not UNSET:
            field_dict["path"] = path
        if test_data is not UNSET:
            field_dict["testData"] = test_data
        if value is not UNSET:
            field_dict["value"] = value
        if variable is not UNSET:
            field_dict["variable"] = variable

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = TransactionalCheckFixKind(d.pop("kind"))




        label = d.pop("label")

        path = cast(list[int], d.pop("path", UNSET))


        test_data = d.pop("testData", UNSET)

        value = d.pop("value", UNSET)

        variable = d.pop("variable", UNSET)

        transactional_check_fix = cls(
            kind=kind,
            label=label,
            path=path,
            test_data=test_data,
            value=value,
            variable=variable,
        )

        return transactional_check_fix

