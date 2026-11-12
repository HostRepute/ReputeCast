from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.transactional_ref import TransactionalRef





T = TypeVar("T", bound="TransactionalFailoverState")



@_attrs_define
class TransactionalFailoverState:
    """ 
        Attributes:
            from_ (TransactionalRef):
            lost (int):
            messages (int):
            recovering (bool):
            since (datetime.datetime):
            to (TransactionalRef):
            code (str | Unset):
            median_ms (int | Unset):
     """

    from_: TransactionalRef
    lost: int
    messages: int
    recovering: bool
    since: datetime.datetime
    to: TransactionalRef
    code: str | Unset = UNSET
    median_ms: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        from_ = self.from_.to_dict()

        lost = self.lost

        messages = self.messages

        recovering = self.recovering

        since = self.since.isoformat()

        to = self.to.to_dict()

        code = self.code

        median_ms = self.median_ms


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "lost": lost,
            "messages": messages,
            "recovering": recovering,
            "since": since,
            "to": to,
        })
        if code is not UNSET:
            field_dict["code"] = code
        if median_ms is not UNSET:
            field_dict["medianMs"] = median_ms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_ref import TransactionalRef # noqa: PLC0415
        d = dict(src_dict)
        from_ = TransactionalRef.from_dict(d.pop("from"))




        lost = d.pop("lost")

        messages = d.pop("messages")

        recovering = d.pop("recovering")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        to = TransactionalRef.from_dict(d.pop("to"))




        code = d.pop("code", UNSET)

        median_ms = d.pop("medianMs", UNSET)

        transactional_failover_state = cls(
            from_=from_,
            lost=lost,
            messages=messages,
            recovering=recovering,
            since=since,
            to=to,
            code=code,
            median_ms=median_ms,
        )

        return transactional_failover_state

