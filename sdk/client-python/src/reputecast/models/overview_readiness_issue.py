from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.overview_readiness_issue_status import OverviewReadinessIssueStatus






T = TypeVar("T", bound="OverviewReadinessIssue")



@_attrs_define
class OverviewReadinessIssue:
    """ 
        Attributes:
            status (OverviewReadinessIssueStatus):
            title (str):
     """

    status: OverviewReadinessIssueStatus
    title: str





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = OverviewReadinessIssueStatus(d.pop("status"))




        title = d.pop("title")

        overview_readiness_issue = cls(
            status=status,
            title=title,
        )

        return overview_readiness_issue

