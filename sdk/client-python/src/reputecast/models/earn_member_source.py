from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_member_source_kind import EarnMemberSourceKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="EarnMemberSource")



@_attrs_define
class EarnMemberSource:
    """ 
        Attributes:
            kind (EarnMemberSourceKind):
            label (str):
            campaign_id (str | Unset):
            granted_by (str | Unset):
            import_id (str | Unset):
            platform (str | Unset):
            reason (str | Unset):
     """

    kind: EarnMemberSourceKind
    label: str
    campaign_id: str | Unset = UNSET
    granted_by: str | Unset = UNSET
    import_id: str | Unset = UNSET
    platform: str | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        campaign_id = self.campaign_id

        granted_by = self.granted_by

        import_id = self.import_id

        platform = self.platform

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "label": label,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if granted_by is not UNSET:
            field_dict["grantedBy"] = granted_by
        if import_id is not UNSET:
            field_dict["importId"] = import_id
        if platform is not UNSET:
            field_dict["platform"] = platform
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = EarnMemberSourceKind(d.pop("kind"))




        label = d.pop("label")

        campaign_id = d.pop("campaignId", UNSET)

        granted_by = d.pop("grantedBy", UNSET)

        import_id = d.pop("importId", UNSET)

        platform = d.pop("platform", UNSET)

        reason = d.pop("reason", UNSET)

        earn_member_source = cls(
            kind=kind,
            label=label,
            campaign_id=campaign_id,
            granted_by=granted_by,
            import_id=import_id,
            platform=platform,
            reason=reason,
        )

        return earn_member_source

