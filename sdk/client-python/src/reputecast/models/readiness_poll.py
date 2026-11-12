from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_progress import ReadinessProgress
  from ..models.readiness_run_failure import ReadinessRunFailure
  from ..models.readiness_run_ref import ReadinessRunRef





T = TypeVar("T", bound="ReadinessPoll")



@_attrs_define
class ReadinessPoll:
    """ 
        Attributes:
            failed (ReadinessRunFailure | Unset):
            latest (ReadinessRunRef | Unset):
            running (ReadinessProgress | Unset):
     """

    failed: ReadinessRunFailure | Unset = UNSET
    latest: ReadinessRunRef | Unset = UNSET
    running: ReadinessProgress | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_progress import ReadinessProgress # noqa: PLC0415
        from ..models.readiness_run_failure import ReadinessRunFailure # noqa: PLC0415
        from ..models.readiness_run_ref import ReadinessRunRef # noqa: PLC0415
        failed: dict[str, Any] | Unset = UNSET
        if not isinstance(self.failed, Unset):
            failed = self.failed.to_dict()

        latest: dict[str, Any] | Unset = UNSET
        if not isinstance(self.latest, Unset):
            latest = self.latest.to_dict()

        running: dict[str, Any] | Unset = UNSET
        if not isinstance(self.running, Unset):
            running = self.running.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if failed is not UNSET:
            field_dict["failed"] = failed
        if latest is not UNSET:
            field_dict["latest"] = latest
        if running is not UNSET:
            field_dict["running"] = running

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_progress import ReadinessProgress # noqa: PLC0415
        from ..models.readiness_run_failure import ReadinessRunFailure # noqa: PLC0415
        from ..models.readiness_run_ref import ReadinessRunRef # noqa: PLC0415
        d = dict(src_dict)
        _failed = d.pop("failed", UNSET)
        failed: ReadinessRunFailure | Unset
        if isinstance(_failed,  Unset):
            failed = UNSET
        else:
            failed = ReadinessRunFailure.from_dict(_failed)




        _latest = d.pop("latest", UNSET)
        latest: ReadinessRunRef | Unset
        if isinstance(_latest,  Unset):
            latest = UNSET
        else:
            latest = ReadinessRunRef.from_dict(_latest)




        _running = d.pop("running", UNSET)
        running: ReadinessProgress | Unset
        if isinstance(_running,  Unset):
            running = UNSET
        else:
            running = ReadinessProgress.from_dict(_running)




        readiness_poll = cls(
            failed=failed,
            latest=latest,
            running=running,
        )

        return readiness_poll

