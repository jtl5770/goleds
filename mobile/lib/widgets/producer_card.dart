import 'package:flutter/material.dart';

class ProducerCard extends StatelessWidget {
  final String title;
  final IconData icon;
  final bool isEnabled;
  final VoidCallback onToggle;
  final VoidCallback onTap;
  final Color accentColor;

  const ProducerCard({
    super.key,
    required this.title,
    required this.icon,
    required this.isEnabled,
    required this.onToggle,
    required this.onTap,
    this.accentColor = Colors.cyanAccent,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 300),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: BorderRadius.circular(16),
          border: isEnabled
              ? Border.all(color: accentColor.withValues(alpha: 0.6), width: 2)
              : Border.all(color: Colors.transparent, width: 2),
          boxShadow: isEnabled
              ? [
                  BoxShadow(
                    color: accentColor.withValues(alpha: 0.3),
                    blurRadius: 12,
                    spreadRadius: 2,
                  )
                ]
              : [],
        ),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 12.0),
          child: Row(
            children: [
              Icon(
                icon,
                size: 32,
                color: isEnabled ? accentColor : Colors.grey,
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                        color: isEnabled ? Colors.white : Colors.grey,
                      ),
                    ),
                    Text(
                      isEnabled ? 'ACTIVE' : 'OFFLINE',
                      style: TextStyle(
                        fontSize: 10,
                        color: isEnabled ? accentColor : Colors.grey.shade700,
                        letterSpacing: 1.2,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
              Switch(
                value: isEnabled,
                onChanged: (val) => onToggle(),
                activeTrackColor: accentColor.withValues(alpha: 0.5),
                activeThumbColor: accentColor,
              ),
            ],
          ),
        ),
      ),
    );
  }
}