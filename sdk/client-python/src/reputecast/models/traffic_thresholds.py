from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.traffic_thresholds_block_action import TrafficThresholdsBlockAction
from ..models.traffic_thresholds_complaint_action import TrafficThresholdsComplaintAction
from ..models.traffic_thresholds_deferral_action import TrafficThresholdsDeferralAction
from ..models.traffic_thresholds_deferral_slow_action import TrafficThresholdsDeferralSlowAction
from ..models.traffic_thresholds_deferral_slow_window import TrafficThresholdsDeferralSlowWindow
from ..models.traffic_thresholds_deferral_window import TrafficThresholdsDeferralWindow
from ..models.traffic_thresholds_unknown_pause_action import TrafficThresholdsUnknownPauseAction
from ..models.traffic_thresholds_unknown_warn_action import TrafficThresholdsUnknownWarnAction






T = TypeVar("T", bound="TrafficThresholds")



@_attrs_define
class TrafficThresholds:
    """ 
        Attributes:
            block_action (TrafficThresholdsBlockAction):
            block_bounce (float):
            complaint (float): Complaints per mailbox provider, where they're reported
            complaint_action (TrafficThresholdsComplaintAction):
            deferral (float):
            deferral_action (TrafficThresholdsDeferralAction):
            deferral_slow (float):
            deferral_slow_action (TrafficThresholdsDeferralSlowAction):
            deferral_slow_window (TrafficThresholdsDeferralSlowWindow):
            deferral_window (TrafficThresholdsDeferralWindow):
            unknown_pause (float):
            unknown_pause_action (TrafficThresholdsUnknownPauseAction):
            unknown_warn (float):
            unknown_warn_action (TrafficThresholdsUnknownWarnAction):
     """

    block_action: TrafficThresholdsBlockAction
    block_bounce: float
    complaint: float
    complaint_action: TrafficThresholdsComplaintAction
    deferral: float
    deferral_action: TrafficThresholdsDeferralAction
    deferral_slow: float
    deferral_slow_action: TrafficThresholdsDeferralSlowAction
    deferral_slow_window: TrafficThresholdsDeferralSlowWindow
    deferral_window: TrafficThresholdsDeferralWindow
    unknown_pause: float
    unknown_pause_action: TrafficThresholdsUnknownPauseAction
    unknown_warn: float
    unknown_warn_action: TrafficThresholdsUnknownWarnAction





    def to_dict(self) -> dict[str, Any]:
        block_action = self.block_action.value

        block_bounce = self.block_bounce

        complaint = self.complaint

        complaint_action = self.complaint_action.value

        deferral = self.deferral

        deferral_action = self.deferral_action.value

        deferral_slow = self.deferral_slow

        deferral_slow_action = self.deferral_slow_action.value

        deferral_slow_window = self.deferral_slow_window.value

        deferral_window = self.deferral_window.value

        unknown_pause = self.unknown_pause

        unknown_pause_action = self.unknown_pause_action.value

        unknown_warn = self.unknown_warn

        unknown_warn_action = self.unknown_warn_action.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blockAction": block_action,
            "blockBounce": block_bounce,
            "complaint": complaint,
            "complaintAction": complaint_action,
            "deferral": deferral,
            "deferralAction": deferral_action,
            "deferralSlow": deferral_slow,
            "deferralSlowAction": deferral_slow_action,
            "deferralSlowWindow": deferral_slow_window,
            "deferralWindow": deferral_window,
            "unknownPause": unknown_pause,
            "unknownPauseAction": unknown_pause_action,
            "unknownWarn": unknown_warn,
            "unknownWarnAction": unknown_warn_action,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        block_action = TrafficThresholdsBlockAction(d.pop("blockAction"))




        block_bounce = d.pop("blockBounce")

        complaint = d.pop("complaint")

        complaint_action = TrafficThresholdsComplaintAction(d.pop("complaintAction"))




        deferral = d.pop("deferral")

        deferral_action = TrafficThresholdsDeferralAction(d.pop("deferralAction"))




        deferral_slow = d.pop("deferralSlow")

        deferral_slow_action = TrafficThresholdsDeferralSlowAction(d.pop("deferralSlowAction"))




        deferral_slow_window = TrafficThresholdsDeferralSlowWindow(d.pop("deferralSlowWindow"))




        deferral_window = TrafficThresholdsDeferralWindow(d.pop("deferralWindow"))




        unknown_pause = d.pop("unknownPause")

        unknown_pause_action = TrafficThresholdsUnknownPauseAction(d.pop("unknownPauseAction"))




        unknown_warn = d.pop("unknownWarn")

        unknown_warn_action = TrafficThresholdsUnknownWarnAction(d.pop("unknownWarnAction"))




        traffic_thresholds = cls(
            block_action=block_action,
            block_bounce=block_bounce,
            complaint=complaint,
            complaint_action=complaint_action,
            deferral=deferral,
            deferral_action=deferral_action,
            deferral_slow=deferral_slow,
            deferral_slow_action=deferral_slow_action,
            deferral_slow_window=deferral_slow_window,
            deferral_window=deferral_window,
            unknown_pause=unknown_pause,
            unknown_pause_action=unknown_pause_action,
            unknown_warn=unknown_warn,
            unknown_warn_action=unknown_warn_action,
        )

        return traffic_thresholds

