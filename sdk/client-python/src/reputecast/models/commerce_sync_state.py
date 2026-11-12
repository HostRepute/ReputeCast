from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_sync_state_revoked_reason import CommerceSyncStateRevokedReason
from ..models.commerce_sync_state_state import CommerceSyncStateState
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.commerce_import import CommerceImport





T = TypeVar("T", bound="CommerceSyncState")



@_attrs_define
class CommerceSyncState:
    """ 
        Attributes:
            missing (list[str]):
            revoked_reason (CommerceSyncStateRevokedReason):
            state (CommerceSyncStateState):
            disconnected_at (datetime.datetime | Unset):
            disconnected_by (str | Unset):
            import_ (CommerceImport | Unset):
            last_event_at (datetime.datetime | Unset):
            missing_since (datetime.datetime | Unset):
            rate_limited_until (datetime.datetime | Unset):
            revoked_at (datetime.datetime | Unset):
     """

    missing: list[str]
    revoked_reason: CommerceSyncStateRevokedReason
    state: CommerceSyncStateState
    disconnected_at: datetime.datetime | Unset = UNSET
    disconnected_by: str | Unset = UNSET
    import_: CommerceImport | Unset = UNSET
    last_event_at: datetime.datetime | Unset = UNSET
    missing_since: datetime.datetime | Unset = UNSET
    rate_limited_until: datetime.datetime | Unset = UNSET
    revoked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.commerce_import import CommerceImport # noqa: PLC0415
        missing = self.missing



        revoked_reason = self.revoked_reason.value

        state = self.state.value

        disconnected_at: str | Unset = UNSET
        if not isinstance(self.disconnected_at, Unset):
            disconnected_at = self.disconnected_at.isoformat()

        disconnected_by = self.disconnected_by

        import_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.import_, Unset):
            import_ = self.import_.to_dict()

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        missing_since: str | Unset = UNSET
        if not isinstance(self.missing_since, Unset):
            missing_since = self.missing_since.isoformat()

        rate_limited_until: str | Unset = UNSET
        if not isinstance(self.rate_limited_until, Unset):
            rate_limited_until = self.rate_limited_until.isoformat()

        revoked_at: str | Unset = UNSET
        if not isinstance(self.revoked_at, Unset):
            revoked_at = self.revoked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "missing": missing,
            "revokedReason": revoked_reason,
            "state": state,
        })
        if disconnected_at is not UNSET:
            field_dict["disconnectedAt"] = disconnected_at
        if disconnected_by is not UNSET:
            field_dict["disconnectedBy"] = disconnected_by
        if import_ is not UNSET:
            field_dict["import"] = import_
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if missing_since is not UNSET:
            field_dict["missingSince"] = missing_since
        if rate_limited_until is not UNSET:
            field_dict["rateLimitedUntil"] = rate_limited_until
        if revoked_at is not UNSET:
            field_dict["revokedAt"] = revoked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.commerce_import import CommerceImport # noqa: PLC0415
        d = dict(src_dict)
        missing = cast(list[str], d.pop("missing"))


        revoked_reason = CommerceSyncStateRevokedReason(d.pop("revokedReason"))




        state = CommerceSyncStateState(d.pop("state"))




        _disconnected_at = d.pop("disconnectedAt", UNSET)
        disconnected_at: datetime.datetime | Unset
        if isinstance(_disconnected_at,  Unset):
            disconnected_at = UNSET
        else:
            disconnected_at = datetime.datetime.fromisoformat(_disconnected_at)




        disconnected_by = d.pop("disconnectedBy", UNSET)

        _import_ = d.pop("import", UNSET)
        import_: CommerceImport | Unset
        if isinstance(_import_,  Unset):
            import_ = UNSET
        else:
            import_ = CommerceImport.from_dict(_import_)




        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        _missing_since = d.pop("missingSince", UNSET)
        missing_since: datetime.datetime | Unset
        if isinstance(_missing_since,  Unset):
            missing_since = UNSET
        else:
            missing_since = datetime.datetime.fromisoformat(_missing_since)




        _rate_limited_until = d.pop("rateLimitedUntil", UNSET)
        rate_limited_until: datetime.datetime | Unset
        if isinstance(_rate_limited_until,  Unset):
            rate_limited_until = UNSET
        else:
            rate_limited_until = datetime.datetime.fromisoformat(_rate_limited_until)




        _revoked_at = d.pop("revokedAt", UNSET)
        revoked_at: datetime.datetime | Unset
        if isinstance(_revoked_at,  Unset):
            revoked_at = UNSET
        else:
            revoked_at = datetime.datetime.fromisoformat(_revoked_at)




        commerce_sync_state = cls(
            missing=missing,
            revoked_reason=revoked_reason,
            state=state,
            disconnected_at=disconnected_at,
            disconnected_by=disconnected_by,
            import_=import_,
            last_event_at=last_event_at,
            missing_since=missing_since,
            rate_limited_until=rate_limited_until,
            revoked_at=revoked_at,
        )

        return commerce_sync_state

