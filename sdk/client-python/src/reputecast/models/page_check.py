from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_check_key import PageCheckKey
from ..models.page_check_status import PageCheckStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PageCheck")



@_attrs_define
class PageCheck:
    """ 
        Attributes:
            key (PageCheckKey):
            status (PageCheckStatus):
            account (str | Unset):
            at (datetime.datetime | Unset):
            by (str | Unset):
            count (int | Unset):
            detail (str | Unset):
            http_status (int | Unset):
            mode (str | Unset):
            name (str | Unset):
            path (str | Unset):
            problem (str | Unset):
            pwyw (bool | Unset):
            ref (str | Unset):
     """

    key: PageCheckKey
    status: PageCheckStatus
    account: str | Unset = UNSET
    at: datetime.datetime | Unset = UNSET
    by: str | Unset = UNSET
    count: int | Unset = UNSET
    detail: str | Unset = UNSET
    http_status: int | Unset = UNSET
    mode: str | Unset = UNSET
    name: str | Unset = UNSET
    path: str | Unset = UNSET
    problem: str | Unset = UNSET
    pwyw: bool | Unset = UNSET
    ref: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        status = self.status.value

        account = self.account

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        by = self.by

        count = self.count

        detail = self.detail

        http_status = self.http_status

        mode = self.mode

        name = self.name

        path = self.path

        problem = self.problem

        pwyw = self.pwyw

        ref = self.ref


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "status": status,
        })
        if account is not UNSET:
            field_dict["account"] = account
        if at is not UNSET:
            field_dict["at"] = at
        if by is not UNSET:
            field_dict["by"] = by
        if count is not UNSET:
            field_dict["count"] = count
        if detail is not UNSET:
            field_dict["detail"] = detail
        if http_status is not UNSET:
            field_dict["httpStatus"] = http_status
        if mode is not UNSET:
            field_dict["mode"] = mode
        if name is not UNSET:
            field_dict["name"] = name
        if path is not UNSET:
            field_dict["path"] = path
        if problem is not UNSET:
            field_dict["problem"] = problem
        if pwyw is not UNSET:
            field_dict["pwyw"] = pwyw
        if ref is not UNSET:
            field_dict["ref"] = ref

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = PageCheckKey(d.pop("key"))




        status = PageCheckStatus(d.pop("status"))




        account = d.pop("account", UNSET)

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        by = d.pop("by", UNSET)

        count = d.pop("count", UNSET)

        detail = d.pop("detail", UNSET)

        http_status = d.pop("httpStatus", UNSET)

        mode = d.pop("mode", UNSET)

        name = d.pop("name", UNSET)

        path = d.pop("path", UNSET)

        problem = d.pop("problem", UNSET)

        pwyw = d.pop("pwyw", UNSET)

        ref = d.pop("ref", UNSET)

        page_check = cls(
            key=key,
            status=status,
            account=account,
            at=at,
            by=by,
            count=count,
            detail=detail,
            http_status=http_status,
            mode=mode,
            name=name,
            path=path,
            problem=problem,
            pwyw=pwyw,
            ref=ref,
        )

        return page_check

