from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_check_group import TransactionalCheckGroup
from ..models.transactional_check_severity import TransactionalCheckSeverity
from ..models.transactional_check_where import TransactionalCheckWhere
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_check_fix import TransactionalCheckFix





T = TypeVar("T", bound="TransactionalCheck")



@_attrs_define
class TransactionalCheck:
    """ 
        Attributes:
            fixes (list[TransactionalCheckFix]):
            group (TransactionalCheckGroup):
            id (str):  Example: variables.undeclared.
            path (list[int]):
            severity (TransactionalCheckSeverity):
            title (str):
            detail (str | Unset):
            line (int | Unset):
            test_data (str | Unset):
            variable (str | Unset):
            where (TransactionalCheckWhere | Unset):
     """

    fixes: list[TransactionalCheckFix]
    group: TransactionalCheckGroup
    id: str
    path: list[int]
    severity: TransactionalCheckSeverity
    title: str
    detail: str | Unset = UNSET
    line: int | Unset = UNSET
    test_data: str | Unset = UNSET
    variable: str | Unset = UNSET
    where: TransactionalCheckWhere | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_check_fix import TransactionalCheckFix # noqa: PLC0415
        fixes = []
        for fixes_item_data in self.fixes:
            fixes_item = fixes_item_data.to_dict()
            fixes.append(fixes_item)



        group = self.group.value

        id = self.id

        path = self.path



        severity = self.severity.value

        title = self.title

        detail = self.detail

        line = self.line

        test_data = self.test_data

        variable = self.variable

        where: str | Unset = UNSET
        if not isinstance(self.where, Unset):
            where = self.where.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fixes": fixes,
            "group": group,
            "id": id,
            "path": path,
            "severity": severity,
            "title": title,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if line is not UNSET:
            field_dict["line"] = line
        if test_data is not UNSET:
            field_dict["testData"] = test_data
        if variable is not UNSET:
            field_dict["variable"] = variable
        if where is not UNSET:
            field_dict["where"] = where

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_check_fix import TransactionalCheckFix # noqa: PLC0415
        d = dict(src_dict)
        fixes = []
        _fixes = d.pop("fixes")
        for fixes_item_data in (_fixes):
            fixes_item = TransactionalCheckFix.from_dict(fixes_item_data)



            fixes.append(fixes_item)


        group = TransactionalCheckGroup(d.pop("group"))




        id = d.pop("id")

        path = cast(list[int], d.pop("path"))


        severity = TransactionalCheckSeverity(d.pop("severity"))




        title = d.pop("title")

        detail = d.pop("detail", UNSET)

        line = d.pop("line", UNSET)

        test_data = d.pop("testData", UNSET)

        variable = d.pop("variable", UNSET)

        _where = d.pop("where", UNSET)
        where: TransactionalCheckWhere | Unset
        if isinstance(_where,  Unset):
            where = UNSET
        else:
            where = TransactionalCheckWhere(_where)




        transactional_check = cls(
            fixes=fixes,
            group=group,
            id=id,
            path=path,
            severity=severity,
            title=title,
            detail=detail,
            line=line,
            test_data=test_data,
            variable=variable,
            where=where,
        )

        return transactional_check

