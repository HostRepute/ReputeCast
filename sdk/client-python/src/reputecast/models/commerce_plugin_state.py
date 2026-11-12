from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommercePluginState")



@_attrs_define
class CommercePluginState:
    """ 
        Attributes:
            checkouts (bool):
            consent (bool):
            found (bool):
            host (str):
            skipped (bool):
            checked_at (datetime.datetime | Unset):
            site_id (str | Unset):
            version (str | Unset):
     """

    checkouts: bool
    consent: bool
    found: bool
    host: str
    skipped: bool
    checked_at: datetime.datetime | Unset = UNSET
    site_id: str | Unset = UNSET
    version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checkouts = self.checkouts

        consent = self.consent

        found = self.found

        host = self.host

        skipped = self.skipped

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        site_id = self.site_id

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkouts": checkouts,
            "consent": consent,
            "found": found,
            "host": host,
            "skipped": skipped,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if site_id is not UNSET:
            field_dict["siteId"] = site_id
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checkouts = d.pop("checkouts")

        consent = d.pop("consent")

        found = d.pop("found")

        host = d.pop("host")

        skipped = d.pop("skipped")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        site_id = d.pop("siteId", UNSET)

        version = d.pop("version", UNSET)

        commerce_plugin_state = cls(
            checkouts=checkouts,
            consent=consent,
            found=found,
            host=host,
            skipped=skipped,
            checked_at=checked_at,
            site_id=site_id,
            version=version,
        )

        return commerce_plugin_state

