from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_send_result_status import TransactionalSendResultStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalSendResult")



@_attrs_define
class TransactionalSendResult:
    """ 
        Attributes:
            id (str):  Example: 0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44.
            status (TransactionalSendResultStatus): not_delivered: a test-mode key's message to an address that isn't
                allowed
            detail (str | Unset):
            send_at (datetime.datetime | Unset):
            test (bool | Unset): Sent with a test-mode key
     """

    id: str
    status: TransactionalSendResultStatus
    detail: str | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    test: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        status = self.status.value

        detail = self.detail

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        test = self.test


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if send_at is not UNSET:
            field_dict["send_at"] = send_at
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        status = TransactionalSendResultStatus(d.pop("status"))




        detail = d.pop("detail", UNSET)

        _send_at = d.pop("send_at", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        test = d.pop("test", UNSET)

        transactional_send_result = cls(
            id=id,
            status=status,
            detail=detail,
            send_at=send_at,
            test=test,
        )

        return transactional_send_result

