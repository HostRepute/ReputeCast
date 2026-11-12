from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_blocked_retry import ReportBlockedRetry
  from ..models.report_bounce_diagnostic import ReportBounceDiagnostic





T = TypeVar("T", bound="ReportBounceReview")



@_attrs_define
class ReportBounceReview:
    """ 
        Attributes:
            count (int):
            diagnostics (list[ReportBounceDiagnostic]):
            label (str):
            type_ (str):
            unsuppressed (int):
            retry (ReportBlockedRetry | Unset):
     """

    count: int
    diagnostics: list[ReportBounceDiagnostic]
    label: str
    type_: str
    unsuppressed: int
    retry: ReportBlockedRetry | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_blocked_retry import ReportBlockedRetry # noqa: PLC0415
        from ..models.report_bounce_diagnostic import ReportBounceDiagnostic # noqa: PLC0415
        count = self.count

        diagnostics = []
        for diagnostics_item_data in self.diagnostics:
            diagnostics_item = diagnostics_item_data.to_dict()
            diagnostics.append(diagnostics_item)



        label = self.label

        type_ = self.type_

        unsuppressed = self.unsuppressed

        retry: dict[str, Any] | Unset = UNSET
        if not isinstance(self.retry, Unset):
            retry = self.retry.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "diagnostics": diagnostics,
            "label": label,
            "type": type_,
            "unsuppressed": unsuppressed,
        })
        if retry is not UNSET:
            field_dict["retry"] = retry

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_blocked_retry import ReportBlockedRetry # noqa: PLC0415
        from ..models.report_bounce_diagnostic import ReportBounceDiagnostic # noqa: PLC0415
        d = dict(src_dict)
        count = d.pop("count")

        diagnostics = []
        _diagnostics = d.pop("diagnostics")
        for diagnostics_item_data in (_diagnostics):
            diagnostics_item = ReportBounceDiagnostic.from_dict(diagnostics_item_data)



            diagnostics.append(diagnostics_item)


        label = d.pop("label")

        type_ = d.pop("type")

        unsuppressed = d.pop("unsuppressed")

        _retry = d.pop("retry", UNSET)
        retry: ReportBlockedRetry | Unset
        if isinstance(_retry,  Unset):
            retry = UNSET
        else:
            retry = ReportBlockedRetry.from_dict(_retry)




        report_bounce_review = cls(
            count=count,
            diagnostics=diagnostics,
            label=label,
            type_=type_,
            unsuppressed=unsuppressed,
            retry=retry,
        )

        return report_bounce_review

