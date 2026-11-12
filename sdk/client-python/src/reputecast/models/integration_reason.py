from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_reason_code import IntegrationReasonCode
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.word_press_check import WordPressCheck





T = TypeVar("T", bound="IntegrationReason")



@_attrs_define
class IntegrationReason:
    """ 
        Attributes:
            code (IntegrationReasonCode):
            by (str | Unset):
            calls (int | Unset):
            check (WordPressCheck | Unset):
            connection_id (str | Unset):
            failed (int | Unset):
            host (str | Unset):
            next_at (datetime.datetime | Unset):
            rejected (int | Unset):
            retrying (int | Unset):
            since (datetime.datetime | Unset):
            status (int | Unset):
            status_text (str | Unset):
     """

    code: IntegrationReasonCode
    by: str | Unset = UNSET
    calls: int | Unset = UNSET
    check: WordPressCheck | Unset = UNSET
    connection_id: str | Unset = UNSET
    failed: int | Unset = UNSET
    host: str | Unset = UNSET
    next_at: datetime.datetime | Unset = UNSET
    rejected: int | Unset = UNSET
    retrying: int | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    status: int | Unset = UNSET
    status_text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.word_press_check import WordPressCheck # noqa: PLC0415
        code = self.code.value

        by = self.by

        calls = self.calls

        check: dict[str, Any] | Unset = UNSET
        if not isinstance(self.check, Unset):
            check = self.check.to_dict()

        connection_id = self.connection_id

        failed = self.failed

        host = self.host

        next_at: str | Unset = UNSET
        if not isinstance(self.next_at, Unset):
            next_at = self.next_at.isoformat()

        rejected = self.rejected

        retrying = self.retrying

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        status = self.status

        status_text = self.status_text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
        })
        if by is not UNSET:
            field_dict["by"] = by
        if calls is not UNSET:
            field_dict["calls"] = calls
        if check is not UNSET:
            field_dict["check"] = check
        if connection_id is not UNSET:
            field_dict["connectionId"] = connection_id
        if failed is not UNSET:
            field_dict["failed"] = failed
        if host is not UNSET:
            field_dict["host"] = host
        if next_at is not UNSET:
            field_dict["nextAt"] = next_at
        if rejected is not UNSET:
            field_dict["rejected"] = rejected
        if retrying is not UNSET:
            field_dict["retrying"] = retrying
        if since is not UNSET:
            field_dict["since"] = since
        if status is not UNSET:
            field_dict["status"] = status
        if status_text is not UNSET:
            field_dict["statusText"] = status_text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.word_press_check import WordPressCheck # noqa: PLC0415
        d = dict(src_dict)
        code = IntegrationReasonCode(d.pop("code"))




        by = d.pop("by", UNSET)

        calls = d.pop("calls", UNSET)

        _check = d.pop("check", UNSET)
        check: WordPressCheck | Unset
        if isinstance(_check,  Unset):
            check = UNSET
        else:
            check = WordPressCheck.from_dict(_check)




        connection_id = d.pop("connectionId", UNSET)

        failed = d.pop("failed", UNSET)

        host = d.pop("host", UNSET)

        _next_at = d.pop("nextAt", UNSET)
        next_at: datetime.datetime | Unset
        if isinstance(_next_at,  Unset):
            next_at = UNSET
        else:
            next_at = datetime.datetime.fromisoformat(_next_at)




        rejected = d.pop("rejected", UNSET)

        retrying = d.pop("retrying", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        status = d.pop("status", UNSET)

        status_text = d.pop("statusText", UNSET)

        integration_reason = cls(
            code=code,
            by=by,
            calls=calls,
            check=check,
            connection_id=connection_id,
            failed=failed,
            host=host,
            next_at=next_at,
            rejected=rejected,
            retrying=retrying,
            since=since,
            status=status,
            status_text=status_text,
        )

        return integration_reason

