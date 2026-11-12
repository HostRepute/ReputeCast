from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="OverviewRefreshFailure")



@_attrs_define
class OverviewRefreshFailure:
    """ 
        Attributes:
            attempts (list[datetime.datetime]):
            op (str): GetAccount
            code (str | Unset):
            last_success_at (datetime.datetime | Unset):
            message (str | Unset):
            status (int | Unset):
            status_label (str | Unset):
            status_url (str | Unset):
     """

    attempts: list[datetime.datetime]
    op: str
    code: str | Unset = UNSET
    last_success_at: datetime.datetime | Unset = UNSET
    message: str | Unset = UNSET
    status: int | Unset = UNSET
    status_label: str | Unset = UNSET
    status_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        attempts = []
        for attempts_item_data in self.attempts:
            attempts_item = attempts_item_data.isoformat()
            attempts.append(attempts_item)



        op = self.op

        code = self.code

        last_success_at: str | Unset = UNSET
        if not isinstance(self.last_success_at, Unset):
            last_success_at = self.last_success_at.isoformat()

        message = self.message

        status = self.status

        status_label = self.status_label

        status_url = self.status_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attempts": attempts,
            "op": op,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if last_success_at is not UNSET:
            field_dict["lastSuccessAt"] = last_success_at
        if message is not UNSET:
            field_dict["message"] = message
        if status is not UNSET:
            field_dict["status"] = status
        if status_label is not UNSET:
            field_dict["statusLabel"] = status_label
        if status_url is not UNSET:
            field_dict["statusUrl"] = status_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attempts = []
        _attempts = d.pop("attempts")
        for attempts_item_data in (_attempts):
            attempts_item = datetime.datetime.fromisoformat(attempts_item_data)



            attempts.append(attempts_item)


        op = d.pop("op")

        code = d.pop("code", UNSET)

        _last_success_at = d.pop("lastSuccessAt", UNSET)
        last_success_at: datetime.datetime | Unset
        if isinstance(_last_success_at,  Unset):
            last_success_at = UNSET
        else:
            last_success_at = datetime.datetime.fromisoformat(_last_success_at)




        message = d.pop("message", UNSET)

        status = d.pop("status", UNSET)

        status_label = d.pop("statusLabel", UNSET)

        status_url = d.pop("statusUrl", UNSET)

        overview_refresh_failure = cls(
            attempts=attempts,
            op=op,
            code=code,
            last_success_at=last_success_at,
            message=message,
            status=status,
            status_label=status_label,
            status_url=status_url,
        )

        return overview_refresh_failure

