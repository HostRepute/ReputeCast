from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_settings_input_late_hours import DigestSettingsInputLateHours
from ..models.digest_settings_input_mode import DigestSettingsInputMode
from ..models.digest_settings_input_source import DigestSettingsInputSource
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID

if TYPE_CHECKING:
  from ..models.digest_filters import DigestFilters
  from ..models.digest_schedule import DigestSchedule
  from ..models.digest_thresholds_input import DigestThresholdsInput





T = TypeVar("T", bound="DigestSettingsInput")



@_attrs_define
class DigestSettingsInput:
    """ 
        Attributes:
            archive (bool):
            category_filter (bool):
            filters (DigestFilters):
            late_hours (DigestSettingsInputLateHours):
            list_ids (list[str]):
            mode (DigestSettingsInputMode):
            name (str):
            prepare_hours (int):
            schedule (DigestSchedule):
            segment_ids (list[str]):
            source (DigestSettingsInputSource):
            feed_url (str | Unset):
            sender_id (str | Unset):
            site_id (UUID | Unset):
            thresholds (DigestThresholdsInput | Unset):
     """

    archive: bool
    category_filter: bool
    filters: DigestFilters
    late_hours: DigestSettingsInputLateHours
    list_ids: list[str]
    mode: DigestSettingsInputMode
    name: str
    prepare_hours: int
    schedule: DigestSchedule
    segment_ids: list[str]
    source: DigestSettingsInputSource
    feed_url: str | Unset = UNSET
    sender_id: str | Unset = UNSET
    site_id: UUID | Unset = UNSET
    thresholds: DigestThresholdsInput | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_filters import DigestFilters # noqa: PLC0415
        from ..models.digest_schedule import DigestSchedule # noqa: PLC0415
        from ..models.digest_thresholds_input import DigestThresholdsInput # noqa: PLC0415
        archive = self.archive

        category_filter = self.category_filter

        filters = self.filters.to_dict()

        late_hours = self.late_hours.value

        list_ids = self.list_ids



        mode = self.mode.value

        name = self.name

        prepare_hours = self.prepare_hours

        schedule = self.schedule.to_dict()

        segment_ids = self.segment_ids



        source = self.source.value

        feed_url = self.feed_url

        sender_id = self.sender_id

        site_id: str | Unset = UNSET
        if not isinstance(self.site_id, Unset):
            site_id = str(self.site_id)

        thresholds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.thresholds, Unset):
            thresholds = self.thresholds.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archive": archive,
            "categoryFilter": category_filter,
            "filters": filters,
            "lateHours": late_hours,
            "listIds": list_ids,
            "mode": mode,
            "name": name,
            "prepareHours": prepare_hours,
            "schedule": schedule,
            "segmentIds": segment_ids,
            "source": source,
        })
        if feed_url is not UNSET:
            field_dict["feedUrl"] = feed_url
        if sender_id is not UNSET:
            field_dict["senderId"] = sender_id
        if site_id is not UNSET:
            field_dict["siteId"] = site_id
        if thresholds is not UNSET:
            field_dict["thresholds"] = thresholds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_filters import DigestFilters # noqa: PLC0415
        from ..models.digest_schedule import DigestSchedule # noqa: PLC0415
        from ..models.digest_thresholds_input import DigestThresholdsInput # noqa: PLC0415
        d = dict(src_dict)
        archive = d.pop("archive")

        category_filter = d.pop("categoryFilter")

        filters = DigestFilters.from_dict(d.pop("filters"))




        late_hours = DigestSettingsInputLateHours(d.pop("lateHours"))




        list_ids = cast(list[str], d.pop("listIds"))


        mode = DigestSettingsInputMode(d.pop("mode"))




        name = d.pop("name")

        prepare_hours = d.pop("prepareHours")

        schedule = DigestSchedule.from_dict(d.pop("schedule"))




        segment_ids = cast(list[str], d.pop("segmentIds"))


        source = DigestSettingsInputSource(d.pop("source"))




        feed_url = d.pop("feedUrl", UNSET)

        sender_id = d.pop("senderId", UNSET)

        _site_id = d.pop("siteId", UNSET)
        site_id: UUID | Unset
        if isinstance(_site_id,  Unset):
            site_id = UNSET
        else:
            site_id = UUID(_site_id)




        _thresholds = d.pop("thresholds", UNSET)
        thresholds: DigestThresholdsInput | Unset
        if isinstance(_thresholds,  Unset):
            thresholds = UNSET
        else:
            thresholds = DigestThresholdsInput.from_dict(_thresholds)




        digest_settings_input = cls(
            archive=archive,
            category_filter=category_filter,
            filters=filters,
            late_hours=late_hours,
            list_ids=list_ids,
            mode=mode,
            name=name,
            prepare_hours=prepare_hours,
            schedule=schedule,
            segment_ids=segment_ids,
            source=source,
            feed_url=feed_url,
            sender_id=sender_id,
            site_id=site_id,
            thresholds=thresholds,
        )

        return digest_settings_input

