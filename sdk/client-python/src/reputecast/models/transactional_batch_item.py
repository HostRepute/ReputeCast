from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_batch_item_status import TransactionalBatchItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_problem import TransactionalProblem





T = TypeVar("T", bound="TransactionalBatchItem")



@_attrs_define
class TransactionalBatchItem:
    """ 
        Attributes:
            index (int):
            status (TransactionalBatchItemStatus):
            error (TransactionalProblem | Unset):
            id (str | Unset):
            send_at (datetime.datetime | Unset):
            test (bool | Unset):
     """

    index: int
    status: TransactionalBatchItemStatus
    error: TransactionalProblem | Unset = UNSET
    id: str | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    test: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_problem import TransactionalProblem # noqa: PLC0415
        index = self.index

        status = self.status.value

        error: dict[str, Any] | Unset = UNSET
        if not isinstance(self.error, Unset):
            error = self.error.to_dict()

        id = self.id

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        test = self.test


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "index": index,
            "status": status,
        })
        if error is not UNSET:
            field_dict["error"] = error
        if id is not UNSET:
            field_dict["id"] = id
        if send_at is not UNSET:
            field_dict["send_at"] = send_at
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_problem import TransactionalProblem # noqa: PLC0415
        d = dict(src_dict)
        index = d.pop("index")

        status = TransactionalBatchItemStatus(d.pop("status"))




        _error = d.pop("error", UNSET)
        error: TransactionalProblem | Unset
        if isinstance(_error,  Unset):
            error = UNSET
        else:
            error = TransactionalProblem.from_dict(_error)




        id = d.pop("id", UNSET)

        _send_at = d.pop("send_at", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        test = d.pop("test", UNSET)

        transactional_batch_item = cls(
            index=index,
            status=status,
            error=error,
            id=id,
            send_at=send_at,
            test=test,
        )

        return transactional_batch_item

