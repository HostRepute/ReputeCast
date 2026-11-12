from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_test_result_status import PushTestResultStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushTestResult")



@_attrs_define
class PushTestResult:
    """ 
        Attributes:
            at (datetime.datetime):
            browser (str):
            platform (str):
            status (PushTestResultStatus):
            detail (str | Unset):
     """

    at: datetime.datetime
    browser: str
    platform: str
    status: PushTestResultStatus
    detail: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        browser = self.browser

        platform = self.platform

        status = self.status.value

        detail = self.detail


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "browser": browser,
            "platform": platform,
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        browser = d.pop("browser")

        platform = d.pop("platform")

        status = PushTestResultStatus(d.pop("status"))




        detail = d.pop("detail", UNSET)

        push_test_result = cls(
            at=at,
            browser=browser,
            platform=platform,
            status=status,
            detail=detail,
        )

        return push_test_result

